package balancer

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytom/vapor/common"
)

// Backends backend node list
type Backends struct {
	sync.RWMutex
	nodes    []*Backend
	nodesMap map[string]*Backend
}

// NewBackends create a list of backend nodes
func NewBackends() *Backends {
	return &Backends{
		nodes:    make([]*Backend, 0),
		nodesMap: make(map[string]*Backend),
	}
}

// Add add backend node
func (b *Backends) Add(newnode *Backend) bool {
	b.Lock()
	defer b.Unlock()
	return b.add(newnode)
}

// Delete delete backend node
func (b *Backends) Delete(newnode *Backend) bool {
	b.Lock()
	defer b.Unlock()
	return b.delete(newnode)
}

// Get get the backend node by index and name
func (b *Backends) Get(key interface{}) (*Backend, bool) {
	b.RLock()
	defer b.RUnlock()
	return b.get(key)
}

// Len get the length of the backend node
func (b *Backends) Len() int {
	b.RLock()
	defer b.RUnlock()
	return len(b.nodes)
}

// Range traverse back-end nodes
func (b *Backends) Range(f func(index int, backend *Backend) bool) {
	b.RLock()
	defer b.RUnlock()

	for i, node := range b.nodes {
		if !f(i, node) {
			break
		}
	}
}

func (b *Backends) add(newnode *Backend) bool {
	if newnode == nil {
		return false
	}

	if _, ok := b.nodesMap[newnode.URL]; ok {
		return false
	}

	b.nodesMap[newnode.URL] = newnode
	b.nodes = append(b.nodes, newnode)
	return true
}

func (b *Backends) delete(newnode *Backend) bool {
	if newnode == nil {
		return false
	}

	if _, ok := b.nodesMap[newnode.URL]; !ok {
		return false
	}

	delete(b.nodesMap, newnode.URL)

	for i, node := range b.nodes {
		if node.URL == newnode.URL {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			return true
		}
	}

	return false
}

func (b *Backends) get(key interface{}) (*Backend, bool) {
	switch k := key.(type) {
	case int:
		if k >= 0 && k < len(b.nodes) {
			return b.nodes[k], true
		}
	case string:
		if node, ok := b.nodesMap[k]; ok {
			return node, true
		}
	}

	return nil, false
}

// Backend node specific information
type Backend struct {
	URL       string
	State     *State
	Statistic *Statistic
	Cache     *common.Cache
}

// NewBackend creates a Backend.
func NewBackend(url string, cacheSize int) *Backend {
	return &Backend{
		URL: url,
		State: &State{
			alive: true,
		},
		Statistic: &Statistic{},
		Cache:     common.NewCache(cacheSize),
	}
}

// State describes the status information of the node
type State struct {
	alive       bool
	aliveMux    sync.RWMutex
	failures    []*FailureError
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
