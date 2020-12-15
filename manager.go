package balancer

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type manager struct {
	balancers *sync.Map
}

// Manager is a struct map from name to balancer.
var Manager = &manager{
	balancers: new(sync.Map),
}

// Register registers the balancer to the balancer name
func (m *manager) Register(name string, b Balancer) {
	m.balancers.Store(strings.ToLower(name), b)
}

// Get returns the resolver balancer registered with the given name.
func (m *manager) Get(name string) Balancer {
	if val, ok := m.balancers.Load(strings.ToLower(name)); ok {
		return val.(Balancer)
	}

	return nil
}

// Balancer get and create a balancer
func (m *manager) Balancer(opts *Options) (Balancer, error) {
	if val, ok := m.balancers.Load(strings.ToLower(opts.Name)); ok {
		return val.(Balancer), nil
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 30
	}
	client := &http.Client{Timeout: time.Duration(opts.Timeout) * time.Second}

	builder := Get(opts.Type)
	if builder == nil {
		return nil, fmt.Errorf("unknown load balance type: %s", opts.Type)
	}

	loadBalancing := builder.Build(client, opts)
	if loadBalancing == nil {
		return nil, fmt.Errorf("%s load balance failed to build", opts.Type)
	}

	m.balancers.Store(strings.ToLower(opts.Name), loadBalancing)
	return loadBalancing, nil
}

// UpdateOptions update balancer config
func (m *manager) UpdateOptions(optsArr []*Options) error {
	for _, opts := range optsArr {
		balancer := m.Get(opts.Name)
		if balancer == nil {
			continue
		}

		optsMap := make(map[string]*Backend)
		backends := balancer.Backends()
		backends.Lock()
		// add new node
		for _, url := range opts.Urls {
			if len(url) == 0 {
				continue
			}

			backend := NewBackend(url, opts.CacheSize)
			optsMap[url] = backend

			if _, ok := backends.get(url); !ok {
				backends.add(backend)
			}
		}

		// delete old node
		for i := len(backends.nodes) - 1; i >= 0; i-- {
			backend := backends.nodes[i]
			if _, ok := optsMap[backend.URL]; !ok {
				backends.delete(backend)
			}
		}
		backends.Unlock()
	}

	return nil
}
