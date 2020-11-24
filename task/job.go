package task

import "sync"

type Job struct {
	spec       string
	fun        func()
	running    bool
	runningMux sync.Mutex
}

func NewJob(spec string, fun func()) *Job {
	return &Job{
		spec:       spec,
		fun:        fun,
		running:    false,
		runningMux: sync.Mutex{},
	}
}

func (j *Job) Spec() string {
	return j.spec
}

func (j *Job) Run() {
	j.runningMux.Lock()

	if j.running {
		j.runningMux.Unlock()
		return
	}

	j.running = true
	j.runningMux.Unlock()

	j.fun()

	j.runningMux.Lock()
	j.running = false
	j.runningMux.Unlock()
}
