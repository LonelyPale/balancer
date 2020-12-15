package health

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/bytom/blockcenter/balancer"
)

var (
	builders = make(map[string]balancer.DoctorBuilder)
)

func Register(b balancer.DoctorBuilder) {
	builders[strings.ToLower(b.Name())] = b
}

func Unregister(name string) {
	delete(builders, strings.ToLower(name))
}

func Get(name string) balancer.DoctorBuilder {
	if b, ok := builders[strings.ToLower(name)]; ok {
		return b
	}
	return nil
}

const Name = "Default"

func init() {
	Register(newBuilder())
}

type doctorBuilder struct {
	name string
}

func newBuilder() balancer.DoctorBuilder {
	return &doctorBuilder{
		name: Name,
	}
}

func (d *doctorBuilder) Build(ping balancer.PingHandler, backends *balancer.Backends) balancer.Doctor {
	return &doctor{
		backends: backends,
		ping:     ping,
	}
}

func (d *doctorBuilder) Name() string {
	return d.name
}

type doctor struct {
	backends *balancer.Backends
	ping     balancer.PingHandler
}

func (d *doctor) HealthCheck() {
	var wg sync.WaitGroup

	d.backends.Range(func(index int, backend *balancer.Backend) bool {
		if !backend.State.Alive() {
			wg.Add(1)
			go func(backend *balancer.Backend) {
				defer wg.Done()
				var errnum int

				for i := 0; i < 10; i++ {
					if err := d.Ping(backend); err != nil {
						errnum += 1
					}
					time.Sleep(10 * time.Second)
				}

				if errnum == 0 {
					backend.State.SetAlive(true)
				}
			}(backend)
		}

		return true
	})

	wg.Wait()
}

func (d *doctor) Ping(backend *balancer.Backend) (err error) {
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
			err = errors.New("Doctor.Ping(): " + str)
		}
	}()

	if d.ping != nil {
		return d.ping(backend)
	}

	return nil
}

func Done(info balancer.DoneInfo) {
	backend := info.Backend
	resp := info.Response
	err := info.Error
	if backend == nil || backend.State == nil {
		return
	}

	var e error
	if err != nil {
		e = err
	} else if resp != nil && resp.StatusCode >= 400 {
		e = errors.New(resp.Status)
	} else {
		return
	}

	backend.State.AddFail(e)
	if backend.State.Alive() {
		backend.State.HealthCheck(600, 100) //10分钟内最多失败100次，超出后backend标记为不可用
	}
}
