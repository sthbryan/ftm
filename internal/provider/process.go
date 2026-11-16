package provider

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

const stopGrace = 3 * time.Second

type Process struct {
	Cmd       *exec.Cmd
	Cancel    context.CancelFunc
	PublicURL string

	exited   chan struct{}
	waitErr  error
	stopOnce sync.Once
}

func StartProcess(cmd *exec.Cmd, cancel context.CancelFunc) (*Process, error) {
	configureProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	p := &Process{
		Cmd:    cmd,
		Cancel: cancel,
		exited: make(chan struct{}),
	}

	go func() {
		p.waitErr = cmd.Wait()
		close(p.exited)
	}()

	return p, nil
}

func (p *Process) Exited() <-chan struct{} {
	return p.exited
}

func (p *Process) Err() error {
	return p.waitErr
}

func (p *Process) Stop() {
	p.stopOnce.Do(func() {
		terminateGroup(p.Cmd)

		select {
		case <-p.exited:
		case <-time.After(stopGrace):

			killGroup(p.Cmd)
			select {
			case <-p.exited:
			case <-time.After(stopGrace):
			}
		}

		if p.Cancel != nil {
			p.Cancel()
		}
	})
}
