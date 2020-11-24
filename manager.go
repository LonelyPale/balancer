package balancer

import "strings"

type manager struct {
	balancers map[string]Balancer
}

// Manager is a struct map from name to balancer.
var Manager = manager{
	balancers: make(map[string]Balancer),
}

// Register registers the balancer to the balancer name
func (m *manager) Register(name string, b Balancer) {
	m.balancers[strings.ToLower(name)] = b
}

// Get returns the resolver balancer registered with the given name.
func (m *manager) Get(name string) Balancer {
	if b, ok := m.balancers[strings.ToLower(name)]; ok {
		return b
	}
	return nil
}
