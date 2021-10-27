package process

import (
	"container/ring"
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	lock           sync.Mutex
	processes      map[string]*Process
	operateChannel chan interface{}
	cron           *cron.Cron
}

func NewManager() *Manager {
	return &Manager{
		processes:      make(map[string]*Process),
		operateChannel: make(chan interface{}, 1024),
		cron:           cron.New(cron.WithSeconds()),
	}
}

func (m *Manager) List() []*Metadata {
	m.lock.Lock()
	defer m.lock.Unlock()

	metadata := make([]*Metadata, 0, len(m.processes))
	for _, p := range m.processes {
		metadata = append(metadata, p.metadata())
	}
	return metadata
}

func (m *Manager) Operate(operate interface{}, timeout time.Duration) error {
	logrus.WithField("timeout", timeout).WithField("operate", operate).Debug("operate receive")

	tm := time.NewTimer(timeout)
	defer tm.Stop()

	select {
	case <-tm.C:
		return fmt.Errorf("operate timeout operate: %v", operate)
	case m.operateChannel <- operate:
		return nil
	}
}

func (m *Manager) handlerOperate(operate interface{}) {
	switch operate.(type) {
	case *OperateStart:
		opt := operate.(*OperateStart)

		process, err := m.createProcess(opt.Dir, opt.Cmd, opt.Argv, opt.Env, opt.Files, opt.Restart, opt.Cron)
		if err != nil {
			return
		}

		if opt.Cron != "" {
			_, _ = m.cron.AddFunc(opt.Cron, func() {
				m.startProcess(process)
			})
			return
		}

		m.startProcess(process)

	case *OperateKill:
		opt := operate.(*OperateKill)
		process, err := m.searchProcess(opt.UUID)
		if err != nil {
			return
		}

		m.killProcess(process, opt.Prune)

	case *OperateStop:
		opt := operate.(*OperateStop)
		process, err := m.searchProcess(opt.UUID)
		if err != nil {
			return
		}

		m.stopProcess(process, opt.Gracefully, opt.Prune)

	case *OperateRestart:
		opt := operate.(*OperateRestart)
		process, err := m.searchProcess(opt.UUID)
		if err != nil {
			return
		}

		m.restartProcess(process, opt.Gracefully)

	case *OperateSignal:
		opt := operate.(*OperateSignal)
		process, err := m.searchProcess(opt.UUID)
		if err != nil {
			return
		}

		m.signalProcess(process, opt.Signal)
	}
}

func (m *Manager) Run(ctx context.Context) error {
	defer func() {
		m.cron.Stop()

		close(m.operateChannel)
	}()

	m.cron.Start()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case operate := <-m.operateChannel:
			m.handlerOperate(operate)
		}
	}
}

func (m *Manager) createProcess(dir, cmd string, argv, env, files []string, restart bool, cron string) (*Process, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	process := &Process{
		manager: m,
		uuid:    uuid.New().String(),
		attributes: &Attributes{
			dir:     dir,
			cmd:     cmd,
			argv:    argv,
			env:     env,
			files:   files,
			restart: restart,
			cron:    cron,
		},
		process:      nil,
		processState: nil,
		files:        nil,
		events:       ring.New(10),
	}

	m.processes[process.uuid] = process

	return process, nil
}

func (m *Manager) searchProcess(uuid string) (*Process, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	process, ok := m.processes[uuid]
	if !ok {
		return nil, fmt.Errorf("not found process: %s", uuid)
	}

	return process, nil
}

func (m *Manager) removeProcess(uuid string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.processes, uuid)
}

func (m *Manager) startProcess(process *Process) {
	if process.isRunning() {
		return
	}

	process.start()
}

func (m *Manager) killProcess(process *Process, prune bool) {
	defer func() {
		if prune {
			m.removeProcess(process.uuid)
		}
	}()

	if !process.isRunning() {
		return
	}

	process.signal(syscall.SIGKILL)
}

func (m *Manager) stopProcess(process *Process, gracefully time.Duration, prune bool) {
	defer func() {
		if prune {
			m.removeProcess(process.uuid)
		}
	}()

	if !process.isRunning() {
		return
	}

	m.do(gracefully, func() {
		process.signal(syscall.SIGTERM)
	}, func() {
		process.signal(syscall.SIGKILL)
	})
}

func (m *Manager) restartProcess(process *Process, gracefully time.Duration) {
	m.stopProcess(process, gracefully, false)

	m.startProcess(process)
}

func (m *Manager) signalProcess(process *Process, signal syscall.Signal) {
	if !process.isRunning() {
		return
	}

	process.signal(signal)
}

func (m *Manager) do(timeout time.Duration, handler, failed func()) {
	t := time.NewTimer(timeout)
	defer t.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case <-t.C:
			failed()
		}
	}()

	handler()
}
