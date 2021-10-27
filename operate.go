package process

import (
	"syscall"
	"time"
)

type OperateStart struct {
	Dir     string
	Cmd     string
	Argv    []string
	Env     []string
	Files   []string
	Restart bool
	Cron    string
}

func newOperateStart(dir, cmd string, argv, env, files []string, restart bool, cron string) *OperateStart {
	return &OperateStart{
		Dir:     dir,
		Cmd:     cmd,
		Argv:    argv,
		Env:     env,
		Files:   files,
		Restart: restart,
		Cron:    cron,
	}
}

type OperateKill struct {
	UUID  string
	Prune bool
}

func newOperateKill(uuid string, prune bool) *OperateKill {
	return &OperateKill{
		UUID:  uuid,
		Prune: prune,
	}
}

type OperateStop struct {
	UUID       string
	Gracefully time.Duration
	Prune      bool
}

func newOperateStop(uuid string, gracefully time.Duration, prune bool) *OperateStop {
	return &OperateStop{
		UUID:       uuid,
		Gracefully: gracefully,
		Prune:      prune,
	}
}

type OperateRestart struct {
	UUID       string
	Gracefully time.Duration
}

func newOperateRestart(uuid string, gracefully time.Duration) *OperateRestart {
	return &OperateRestart{
		UUID:       uuid,
		Gracefully: gracefully,
	}
}

type OperateSignal struct {
	UUID   string
	Signal syscall.Signal
}

func newOperateSignal(uuid string, signal syscall.Signal) *OperateSignal {
	return &OperateSignal{
		UUID:   uuid,
		Signal: signal,
	}
}
