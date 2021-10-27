package process

import (
	"container/ring"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

type Attributes struct {
	dir     string
	cmd     string
	argv    []string
	env     []string
	files   []string
	restart bool
	cron    string
}

type Process struct {
	manager      *Manager
	m            sync.Mutex
	uuid         string
	attributes   *Attributes
	process      *os.Process
	processState *os.ProcessState
	files        []*os.File
	events       *ring.Ring
}

func (p *Process) isRunning() bool {
	p.m.Lock()
	defer p.m.Unlock()

	return p.process != nil && p.processState == nil
}

func (p *Process) pushEvent(kind kind, message string) {
	p.events.Value = &event{
		time:    time.Now(),
		kind:    kind,
		message: message,
	}

	p.events = p.events.Next()
}

func (p *Process) openFiles(names ...string) ([]*os.File, error) {
	files := make([]*os.File, 0, len(names))
	for _, file := range names {
		f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (p *Process) closeFiles(files ...*os.File) {
	for _, file := range files {
		_ = file.Close()
	}
}

func (p *Process) start() {
	p.m.Lock()
	defer p.m.Unlock()

	files, err := p.openFiles(p.attributes.files...)
	if err != nil {
		p.pushEvent(eventKindErr, fmt.Sprintf("process: %s, open file: %v, failed: %s", p.uuid, p.attributes.files, err))
		return
	}

	p.files = files

	p.pushEvent(eventKindData, fmt.Sprintf("process: %s, open file: %v success", p.uuid, p.attributes.files))

	process, err := os.StartProcess(p.attributes.cmd, p.attributes.argv, &os.ProcAttr{
		Dir:   p.attributes.dir,
		Env:   p.attributes.env,
		Files: p.files,
		Sys:   nil,
	})
	if err != nil {
		p.pushEvent(eventKindErr, fmt.Sprintf("process: %s, start failed: %s", p.uuid, err))
		return
	}

	p.process = process
	p.processState = nil

	p.pushEvent(eventKindData, fmt.Sprintf("process: %s, start success, pid: %d", p.uuid, process.Pid))

	go func(process *os.Process, files []*os.File) {
		defer func() {
			if p.attributes.restart {
				_ = p.manager.Operate(&OperateRestart{
					UUID:       p.uuid,
					Gracefully: time.Second,
				}, time.Second)
			}
		}()

		processState, err := process.Wait()

		p.m.Lock()
		defer p.m.Unlock()

		defer p.closeFiles(files...)

		if err != nil {
			p.pushEvent(eventKindErr, fmt.Sprintf("process: %s, wait failed: %s", p.uuid, err))
			return
		}

		p.pushEvent(eventKindData, fmt.Sprintf("process: %s, wait success, pid: %d", p.uuid, process.Pid))

		p.processState = processState
	}(process, files)
}

func (p *Process) signal(s syscall.Signal) {
	p.m.Lock()
	defer p.m.Unlock()

	process := p.process

	if process == nil {
		return
	}

	err := process.Signal(s)
	if err != nil {
		p.pushEvent(eventKindErr, fmt.Sprintf("process: %s, signal %s failed: %s", p.uuid, s, err))
		return
	}

	p.pushEvent(eventKindData, fmt.Sprintf("process: %s, signal %s success, pid: %d", p.uuid, s, process.Pid))
}

type Metadata struct {
	UUID     string
	Pid      int
	Alive    bool
	Dir      string
	Cmd      string
	Argv     []string
	Env      []string
	Files    []string
	Restart  bool
	Cron     string
	Events   []string
	ExitCode int
	ExitData string
}

func (p *Process) metadata() *Metadata {
	p.m.Lock()
	defer p.m.Unlock()

	m := &Metadata{
		UUID:    p.uuid,
		Pid:     -1,
		Alive:   false,
		Dir:     p.attributes.dir,
		Cmd:     p.attributes.cmd,
		Argv:    p.attributes.argv,
		Env:     p.attributes.env,
		Files:   p.attributes.files,
		Restart: p.attributes.restart,
		Cron:    p.attributes.cron,
		Events:  nil,
	}

	if p.process != nil {
		m.Pid = p.process.Pid
		m.Alive = p.process.Signal(syscall.Signal(0)) == nil
	}

	if p.processState != nil {
		m.ExitCode = p.processState.ExitCode()
		m.ExitData = p.processState.String()
	}

	for i := 0; i < p.events.Len(); i++ {
		if p.events.Value != nil {
			e := p.events.Value.(*event)
			m.Events = append(m.Events, fmt.Sprintf("%s %s %s", e.time.Format(time.RFC3339), e.kind, e.message))
		}
		p.events = p.events.Next()
	}

	return m
}
