package round_robin

import (
	"errors"
	"sync"

	"github.com/lonelypale/balancer"
	"github.com/lonelypale/balancer/base"
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

func (*rrPickerBuilder) Build(backends []*balancer.Backend) balancer.Picker {
	return &rrPicker{
		backends: backends,
	}
}

type rrPicker struct {
	backends []*balancer.Backend
	current  int
	mux      sync.RWMutex
}

func (p *rrPicker) Pick() (*balancer.Backend, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	length := len(p.backends)
	next := p.current + 1
	l := next + length
	for i := next; i < l; i++ {
		idx := i % length
		if p.backends[idx].State.Alive() {
			p.current = idx
			return p.backends[idx], nil
		}
	}

	return nil, errors.New("Picker.Pick(): No Backend available")
}
