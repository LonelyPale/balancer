package balancer

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// builders is a map from name to balancer builder.
	builders = make(map[string]Builder)
)

// Register registers the balancer builder to the balancer map. b.Name
func Register(b Builder) {
	builders[strings.ToLower(b.Name())] = b
}

// Unregister deletes the balancer with the given name from the balancer map.
func Unregister(name string) {
	delete(builders, strings.ToLower(name))
}

// Get returns the resolver builder registered with the given name.
func Get(name string) Builder {
	if b, ok := builders[strings.ToLower(name)]; ok {
		return b
	}
	return nil
}

// Options contains additional information for Build.
type Options struct {
	Name        string           `json:"name" mapstructure:"name"`           //Balancer name
	Type        string           `json:"type" mapstructure:"type"`           //Picker type
	Timeout     int              `json:"timeout" mapstructure:"timeout"`     //http timeout, Unit: second
	Urls        []string         `json:"urls" mapstructure:"urls"`           //Load node url
	Doctor      DoctorOptions    `json:"doctor" mapstructure:"doctor"`       //Health checker
	Statistic   StatisticOptions `json:"statistic" mapstructure:"statistic"` //Statistics
	DoneHandler DoneHandler      `json:"-"`
	PingHandler PingHandler      `json:"-"`
}

// DoctorOptions contains additional information for Doctor.
type DoctorOptions struct {
	Enable bool   `json:"enable" mapstructure:"enable"` //Whether to enable health check
	Type   string `json:"type" mapstructure:"type"`     //Doctor type
	Spec   string `json:"spec" mapstructure:"spec"`     //Time interval of scheduled tasks
}

// StatisticOptions contains additional information for Statistic.
type StatisticOptions struct {
	Enable bool `json:"enable" mapstructure:"enable"` //Whether to enable statistics
	Port   int  `json:"port" mapstructure:"port"`     //Service port for obtaining statistics
}

// Builder creates a balancer.
type Builder interface {
	Build(client *http.Client, opts *Options) Balancer
	Name() string
}

// Balancer takes input from http, manages Backend, and collects and aggregates
// the connectivity states.
type Balancer interface {
	Do(req *http.Request) (*http.Response, error)
	Backends() []*Backend
	Close()
}

// PickerBuilder creates balancer.Picker.
type PickerBuilder interface {
	Build(backends []*Backend) Picker
}

// Picker is used by http to pick a Backend to send an http.
type Picker interface {
	Pick() (*Backend, error)
}

// DoctorBuilder creates balancer.Doctor.
type DoctorBuilder interface {
	Build(ping PingHandler, backends []*Backend) Doctor
	Name() string
}

// Doctor checks the health of Backend
type Doctor interface {
	HealthCheck()
	Ping(backend *Backend) error
}

// DoneInfo contains additional information for done.
type DoneInfo struct {
	Backend  *Backend
	Response *http.Response
	Error    error
}

// DoneHandler define the specific implementation of Done
type DoneHandler func(info DoneInfo)

// PingHandler define the specific implementation of Ping
type PingHandler func(backend *Backend) error

// Backend node specific information
type Backend struct {
	URL       string
	State     *State
	Statistic *Statistic
}

// NewBackend creates a Backend.
func NewBackend(url string) *Backend {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	return &Backend{
		URL: url,
		State: &State{
			alive: true,
		},
		Statistic: &Statistic{},
	}
}

// State describes the status information of the node
type State struct {
	alive       bool
	failures    []*FailureError
	aliveMux    sync.RWMutex
	failuresMux sync.RWMutex
}

// SetAlive set whether the node is available
func (s *State) SetAlive(alive bool) {
	s.aliveMux.Lock()
	defer s.aliveMux.Unlock()
	s.alive = alive
}

// Alive if the backend is still alive, Alive returns true
func (s *State) Alive() bool {
	s.aliveMux.RLock()
	defer s.aliveMux.RUnlock()
	return s.alive
}

// AddFail add failed error
func (s *State) AddFail(err error) {
	s.failuresMux.Lock()
	defer s.failuresMux.Unlock()
	n := len(s.failures)
	if n == 100 {
		s.failures = s.failures[1:]
	}
	s.failures = append(s.failures, NewFailureError(err))
}

// CleanFail clear outdated errors
func (s *State) CleanFail(interval int64) {
	s.failuresMux.Lock()
	defer s.failuresMux.Unlock()
	now := time.Now().Unix()
	for _, v := range s.failures {
		if now-v.timestamp >= interval {
			s.failures = s.failures[1:]
		} else {
			break
		}
	}
}

// LenFail return number of failures
func (s *State) LenFail() int {
	s.failuresMux.RLock()
	defer s.failuresMux.RUnlock()
	return len(s.failures)
}

// HealthCheck determine whether the node is available
func (s *State) HealthCheck(interval int64, max int) bool {
	s.CleanFail(interval)
	length := s.LenFail()
	if length >= max {
		s.SetAlive(false)
		return false
	}
	return true
}

// FailureError describe the failed error
type FailureError struct {
	timestamp int64
	err       error
}

// NewFailureError creates a fail error.
func NewFailureError(err error) *FailureError {
	return &FailureError{
		timestamp: time.Now().Unix(),
		err:       err,
	}
}

// Error return error message
func (f *FailureError) Error() string {
	return f.err.Error()
}

// Statistic describe statistics
type Statistic struct {
	success uint64
	failure uint64
}

// Success return number of success
func (s *Statistic) Success() uint64 {
	return atomic.LoadUint64(&s.success)
}

// IncSuccess auto-increment success times
func (s *Statistic) IncSuccess() uint64 {
	return atomic.AddUint64(&s.success, 1)
}

// Failure return number of failures
func (s *Statistic) Failure() uint64 {
	return atomic.LoadUint64(&s.failure)
}

// IncFailure auto-increment failure times
func (s *Statistic) IncFailure() uint64 {
	return atomic.AddUint64(&s.failure, 1)
}
