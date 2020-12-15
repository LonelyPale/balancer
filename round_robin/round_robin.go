package round_robin

import (
	"errors"
	"github.com/bytom/blockcenter/balancer"
	"github.com/bytom/blockcenter/balancer/base"
)

// Name is the name of RoundRobin Builder.
const Name = "RoundRobin"

func init() {
	balancer.Register(newBuilder())
}

// newBuilder creates a new roundrobin balancer builder.
func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, &rrPickerBuilder{})
}

type rrPickerBuilder struct{}

func (*rrPickerBuilder) Build(backends *balancer.Backends) balancer.Picker {
	return &rrPicker{
		backends: backends,
	}
}

type rrPicker struct {
	backends *balancer.Backends
	current  int
}

func (p *rrPicker) Pick() (*balancer.Backend, error) {
	p.backends.RLock()
	defer p.backends.RUnlock()

	length := p.backends.Len()
	next := p.current + 1
	l := next + length
	for i := next; i < l; i++ {
		idx := i % length
		if backend, ok := p.backends.Get(idx); ok && backend.State.Alive() {
			p.current = idx
			return backend, nil
		}
	}

	return nil, errors.New("Picker.Pick(): No Backend available")
}
