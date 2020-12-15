package base

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bytom/blockcenter/balancer"
	"github.com/bytom/blockcenter/balancer/health"
	"github.com/bytom/blockcenter/balancer/statistic"
	"github.com/bytom/blockcenter/balancer/task"
)

type baseBuilder struct {
	name          string
	pickerBuilder balancer.PickerBuilder
}

func NewBalancerBuilder(name string, pb balancer.PickerBuilder) balancer.Builder {
	return &baseBuilder{
		name:          name,
		pickerBuilder: pb,
	}
}

func (bb *baseBuilder) Build(client *http.Client, opts *balancer.Options) balancer.Balancer {
	backends := balancer.NewBackends()
	for _, url := range opts.Urls {
		if len(url) == 0 {
			continue
		}
		backends.Add(balancer.NewBackend(url, opts.CacheSize))
	}

	var doctor balancer.Doctor
	if opts.Doctor.Enable {
		doctorBuilder := health.Get(opts.Doctor.Type)
		if doctorBuilder != nil {
			if opts.DoneHandler == nil {
				opts.DoneHandler = health.Done
			}
			if opts.PingHandler == nil {
				opts.PingHandler = health.BytomPing
			}

			doctor = doctorBuilder.Build(opts.PingHandler, backends)

			if len(opts.Doctor.Spec) == 0 {
				opts.Doctor.Spec = "0 */1 * * * ?"
			}
			err := task.Start(task.NewJob(opts.Doctor.Spec, func() {
				doctor.HealthCheck()
			}))
			if err != nil {
				panic(err)
			}
		}
	}

	loadBalancing := &baseBalancer{
		statistic: &opts.Statistic,
		client:    client,
		picker:    bb.pickerBuilder.Build(backends),
		doctor:    doctor,
		done:      opts.DoneHandler,
		backends:  backends,
	}

	if opts.Statistic.Enable {
		go statistic.ServerAndRun(&opts.Statistic)
	}

	return loadBalancing
}

func (bb *baseBuilder) Name() string {
	return bb.name
}

type baseBalancer struct {
	client    *http.Client
	backends  *balancer.Backends
	statistic *balancer.StatisticOptions
	picker    balancer.Picker
	doctor    balancer.Doctor
	done      balancer.DoneHandler
}

func (b *baseBalancer) Pick() (*balancer.Backend, error) {
	backend, err := b.picker.Pick()
	if err != nil {
		return nil, err
	}
	if backend == nil {
		return nil, errors.New("Picker.Pick(): nil backend")
	}
	return backend, nil
}

func (b *baseBalancer) Do(req *http.Request) (resp *http.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			var str string
			switch e := r.(type) {
			case error:
				str = e.Error()
			case string:
				str = e
			default:
				str = "unknown error"
			}
			err = errors.New("Balancer.Do(): " + str)
		}
	}()

	url := req.URL.String()
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return b.client.Do(req)
	}

	backend, err := b.picker.Pick()
	if err != nil {
		return nil, err
	}
	if backend == nil {
		return nil, errors.New("Picker.Pick(): nil backend")
	}

	var newurl string
	if strings.HasPrefix(backend.URL, "http://") || strings.HasPrefix(backend.URL, "https://") {
		newurl = balancer.URLJoin(backend.URL, url)
	} else {
		newurl = balancer.URLJoin("http://", backend.URL, url)
	}

	if req, err = http.NewRequest(req.Method, newurl, req.Body); err != nil {
		return nil, err
	}

	resp, err = b.client.Do(req)

	if b.statistic.Enable {
		if err == nil && (resp != nil && resp.StatusCode < 400) {
			backend.Statistic.IncSuccess()
		} else {
			backend.Statistic.IncFailure()
		}
	}

	if b.done != nil {
		b.done(balancer.DoneInfo{
			Backend:  backend,
			Response: resp,
			Error:    err,
		})
	}

	return resp, err
}

func (b *baseBalancer) Close() {
	if b.doctor != nil {
		task.Stop()
	}
}

func (b *baseBalancer) Backends() *balancer.Backends {
	return b.backends
}
