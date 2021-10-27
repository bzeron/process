package process

import (
	"fmt"
	"strings"
	"syscall"
	"time"
)

type RPC struct {
	manager *Manager
}

func NewRPC(manager *Manager) *RPC {
	return &RPC{
		manager: manager,
	}
}

type ListArgv struct {
	Cmd    string
	Status string
}

type ListReply struct {
	Metadata []map[string]string
}

func (r *RPC) List(argv *ListArgv, reply *ListReply) error {
	metadata := r.manager.List()

	for _, m := range metadata {
		reply.Metadata = append(reply.Metadata, map[string]string{
			"UUID":     fmt.Sprint(m.UUID),
			"Pid":      fmt.Sprint(m.Pid),
			"Alive":    fmt.Sprint(m.Alive),
			"Dir":      fmt.Sprint(m.Dir),
			"Cmd":      fmt.Sprint(m.Cmd),
			"Argv":     strings.Join(m.Argv, "\n"),
			"Env":      strings.Join(m.Env, "\n"),
			"Files":    strings.Join(m.Files, "\n"),
			"Restart":  fmt.Sprint(m.Restart),
			"Cron":     fmt.Sprint(m.Cron),
			"Events":   strings.Join(m.Events, "\n"),
			"ExitCode": fmt.Sprint(m.ExitCode),
			"ExitData": fmt.Sprint(m.ExitData),
		})
	}
	return nil
}

type StartArgv struct {
	Dir     string
	Cmd     string
	Argv    []string
	Env     []string
	Files   []string
	Restart bool
	Cron    string
}

type StartReply struct {
}

func (r *RPC) Start(argv *StartArgv, reply *StartReply) error {
	return r.manager.Operate(newOperateStart(argv.Dir, argv.Cmd, argv.Argv, argv.Env, argv.Files, argv.Restart, argv.Cron), time.Second*10)
}

type KillArgv struct {
	UUID  string
	Prune bool
}

type KillReply struct {
}

func (r *RPC) Kill(argv *KillArgv, reply *KillReply) error {
	return r.manager.Operate(newOperateKill(argv.UUID, argv.Prune), time.Second*10)
}

type StopArgv struct {
	UUID       string
	Gracefully time.Duration
	Prune      bool
}

type StopReply struct {
}

func (r *RPC) Stop(argv *StopArgv, reply *StopReply) error {
	return r.manager.Operate(newOperateStop(argv.UUID, argv.Gracefully, argv.Prune), time.Second*10)
}

type RestartArgv struct {
	UUID       string
	Gracefully time.Duration
}

type RestartReply struct {
}

func (r *RPC) Restart(argv *RestartArgv, reply *RestartReply) error {
	return r.manager.Operate(newOperateRestart(argv.UUID, argv.Gracefully), time.Second*10)
}

type SignalArgv struct {
	UUID   string
	Signal syscall.Signal
}

type SignalReply struct{}

func (r *RPC) Signal(argv *SignalArgv, reply *SignalReply) error {
	return r.manager.Operate(newOperateSignal(argv.UUID, argv.Signal), time.Second*10)
}
