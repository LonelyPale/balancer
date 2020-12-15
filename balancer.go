package balancer

import (
	"net/http"
	"strings"
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
	Name        string           `json:"name" mapstructure:"name"`             //Balancer name
	Type        string           `json:"type" mapstructure:"type"`             //Picker type
	Timeout     int              `json:"timeout" mapstructure:"timeout"`       //http timeout, Unit: second
	CacheSize   int              `json:"cache_size" mapstructure:"cache_size"` //Node cache size
	NetParam    string           `json:"net_param" mapstructure:"net_param"`   //Node net param
	Urls        []string         `json:"urls" mapstructure:"urls"`             //Load node url
	Doctor      DoctorOptions    `json:"doctor" mapstructure:"doctor"`         //Health checker
	Statistic   StatisticOptions `json:"statistic" mapstructure:"statistic"`   //Statistics
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
	Pick() (*Backend, error)
	Backends() *Backends
	Close()
}

// PickerBuilder creates balancer.Picker.
type PickerBuilder interface {
	Build(backends *Backends) Picker
}

// Picker is used by http to pick a Backend to send an http.
type Picker interface {
	Pick() (*Backend, error)
}

// DoctorBuilder creates balancer.Doctor.
type DoctorBuilder interface {
	Build(ping PingHandler, backends *Backends) Doctor
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
