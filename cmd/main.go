package main

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/bzeron/process"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func main() {
	cobra.OnInitialize(func() {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetFormatter(&logrus.JSONFormatter{})
	})

	root := &cobra.Command{
		Use: "process",
	}

	root.AddCommand(
		newServiceCommand(),
		newListCommand(),
		newStartCommand(),
		newKillCommand(),
		newStopCommand(),
		newRestartCommand(),
		newSignalCommand(),
	)

	root.PersistentFlags().String("network", "tcp", "net listen network")
	root.PersistentFlags().String("address", "0.0.0.0:8080", "net listen address")

	cobra.CheckErr(root.Execute())
}

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "service",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			manager := process.NewManager()

			service := process.NewRPC(manager)

			g, ctx := errgroup.WithContext(cmd.Context())

			g.Go(func() error {
				logrus.Debug("manager service run")
				return manager.Run(ctx)
			})

			g.Go(func() error {
				err = rpc.Register(service)
				if err != nil {
					return err
				}

				listener, err := net.Listen(network, address)
				if err != nil {
					return err
				}

				logrus.Debug("rpc service run")

				for ctx.Err() == nil {
					conn, err := listener.Accept()
					if err != nil {
						return err
					}

					go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
				}

				return ctx.Err()
			})

			return g.Wait()
		},
	}

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			conn, err := net.Dial(network, address)
			if err != nil {
				return err
			}

			argv := &process.ListArgv{}

			reply := &process.ListReply{}

			err = rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)).Call("RPC.List", argv, reply)
			if err != nil {
				return err
			}

			if len(reply.Metadata) == 0 {
				return nil
			}

			keys := make(map[string]struct{})

			for _, meta := range reply.Metadata {
				for k := range meta {
					keys[k] = struct{}{}
				}
			}

			headers := make([]string, 0, len(keys))
			for key := range keys {
				headers = append(headers, key)
			}

			sort.Strings(headers)

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader(headers)
			for _, meta := range reply.Metadata {
				values := make([]string, 0, len(keys))
				for _, header := range headers {
					value, ok := meta[header]
					if ok {
						values = append(values, value)
					} else {
						values = append(values, "")
					}
				}
				table.Append(values)
			}

			table.Render()

			return nil
		},
	}

	return cmd
}

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "start",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			dir, err := cmd.Flags().GetString("dir")
			if err != nil {
				return err
			}

			c, err := cmd.Flags().GetString("cmd")
			if err != nil {
				return err
			}

			v, err := cmd.Flags().GetStringSlice("argv")
			if err != nil {
				return err
			}

			env, err := cmd.Flags().GetStringSlice("env")
			if err != nil {
				return err
			}

			files, err := cmd.Flags().GetStringSlice("files")
			if err != nil {
				return err
			}

			restart, err := cmd.Flags().GetBool("restart")
			if err != nil {
				return err
			}

			cron, err := cmd.Flags().GetString("cron")
			if err != nil {
				return err
			}

			conn, err := net.Dial(network, address)
			if err != nil {
				return err
			}

			argv := &process.StartArgv{
				Dir:     dir,
				Cmd:     c,
				Argv:    v,
				Env:     env,
				Files:   files,
				Restart: restart,
				Cron:    cron,
			}

			reply := &process.StartReply{}

			return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)).Call("RPC.Start", argv, reply)
		},
	}

	cmd.Flags().String("dir", "", "dir")
	cmd.Flags().String("cmd", "", "command")
	cmd.Flags().StringSlice("argv", nil, "argv")
	cmd.Flags().StringSlice("env", nil, "env")
	cmd.Flags().StringSlice("files", nil, "files")
	cmd.Flags().Bool("restart", false, "restart")
	cmd.Flags().String("cron", "", "cron")
	cobra.CheckErr(cmd.MarkFlagRequired("dir"))
	cobra.CheckErr(cmd.MarkFlagRequired("cmd"))
	cobra.CheckErr(cmd.MarkFlagRequired("files"))

	return cmd
}

func newKillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "kill",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			uuid, err := cmd.Flags().GetString("uuid")
			if err != nil {
				return err
			}

			prune, err := cmd.Flags().GetBool("prune")
			if err != nil {
				return err
			}

			conn, err := net.Dial(network, address)
			if err != nil {
				return err
			}

			argv := &process.KillArgv{
				UUID:  uuid,
				Prune: prune,
			}

			reply := &process.KillReply{}

			return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)).Call("RPC.Kill", argv, reply)
		},
	}

	cmd.Flags().String("uuid", "", "uuid")
	cmd.Flags().Bool("prune", false, "prune")
	cobra.CheckErr(cmd.MarkFlagRequired("uuid"))

	return cmd
}

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "stop",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			uuid, err := cmd.Flags().GetString("uuid")
			if err != nil {
				return err
			}

			gracefully, err := cmd.Flags().GetDuration("gracefully")
			if err != nil {
				return err
			}

			prune, err := cmd.Flags().GetBool("prune")
			if err != nil {
				return err
			}

			conn, err := net.Dial(network, address)
			if err != nil {
				return err
			}

			argv := &process.StopArgv{
				UUID:       uuid,
				Gracefully: gracefully,
				Prune:      prune,
			}

			reply := &process.StopReply{}

			return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)).Call("RPC.Stop", argv, reply)
		},
	}

	cmd.Flags().String("uuid", "", "uuid")
	cmd.Flags().Duration("gracefully", time.Second*5, "gracefully")
	cmd.Flags().Bool("prune", false, "prune")
	cobra.CheckErr(cmd.MarkFlagRequired("uuid"))

	return cmd
}

func newRestartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "restart",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			uuid, err := cmd.Flags().GetString("uuid")
			if err != nil {
				return err
			}

			gracefully, err := cmd.Flags().GetDuration("gracefully")
			if err != nil {
				return err
			}

			conn, err := net.Dial(network, address)
			if err != nil {
				return err
			}

			argv := &process.RestartArgv{
				UUID:       uuid,
				Gracefully: gracefully,
			}

			reply := &process.RestartReply{}

			return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)).Call("RPC.Restart", argv, reply)
		},
	}

	cmd.Flags().String("uuid", "", "uuid")
	cmd.Flags().Duration("gracefully", time.Second*5, "gracefully")
	cobra.CheckErr(cmd.MarkFlagRequired("uuid"))

	return cmd
}

func newSignalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "signal",
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := cmd.Flags().GetString("network")
			if err != nil {
				return err
			}

			address, err := cmd.Flags().GetString("address")
			if err != nil {
				return err
			}

			uuid, err := cmd.Flags().GetString("uuid")
			if err != nil {
				return err
			}

			argv := &process.SignalArgv{
				UUID:   uuid,
				Signal: syscall.Signal(0),
			}

			signal, err := cmd.Flags().GetString("signal")
			if err != nil {
				return err
			}

			switch strings.ToUpper(signal) {
			case "ABRT":
				argv.Signal = syscall.SIGABRT
			case "ALRM":
				argv.Signal = syscall.SIGALRM
			case "BUS":
				argv.Signal = syscall.SIGBUS
			case "CHLD":
				argv.Signal = syscall.SIGCHLD
			case "CONT":
				argv.Signal = syscall.SIGCONT
			case "EMT":
				argv.Signal = syscall.SIGEMT
			case "FPE":
				argv.Signal = syscall.SIGFPE
			case "HUP":
				argv.Signal = syscall.SIGHUP
			case "ILL":
				argv.Signal = syscall.SIGILL
			case "INFO":
				argv.Signal = syscall.SIGINFO
			case "INT":
				argv.Signal = syscall.SIGINT
			case "IO":
				argv.Signal = syscall.SIGIO
			case "IOT":
				argv.Signal = syscall.SIGIOT
			case "KILL":
				argv.Signal = syscall.SIGKILL
			case "PIPE":
				argv.Signal = syscall.SIGPIPE
			case "PROF":
				argv.Signal = syscall.SIGPROF
			case "QUIT":
				argv.Signal = syscall.SIGQUIT
			case "SEGV":
				argv.Signal = syscall.SIGSEGV
			case "STOP":
				argv.Signal = syscall.SIGSTOP
			case "SYS":
				argv.Signal = syscall.SIGSYS
			case "TERM":
				argv.Signal = syscall.SIGTERM
			case "TRAP":
				argv.Signal = syscall.SIGTRAP
			case "TSTP":
				argv.Signal = syscall.SIGTSTP
			case "TTIN":
				argv.Signal = syscall.SIGTTIN
			case "TTOU":
				argv.Signal = syscall.SIGTTOU
			case "URG":
				argv.Signal = syscall.SIGURG
			case "USR1":
				argv.Signal = syscall.SIGUSR1
			case "USR2":
				argv.Signal = syscall.SIGUSR2
			case "VTALRM":
				argv.Signal = syscall.SIGVTALRM
			case "WINCH":
				argv.Signal = syscall.SIGWINCH
			case "XCPU":
				argv.Signal = syscall.SIGXCPU
			case "XFSZ":
				argv.Signal = syscall.SIGXFSZ
			}

			conn, err := net.Dial(network, address)
			if err != nil {
				return err
			}

			reply := &process.SignalReply{}

			return rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)).Call("RPC.Signal", argv, reply)
		},
	}

	cmd.Flags().String("uuid", "", "uuid")
	cmd.Flags().String("signal", "", "signal")
	cobra.CheckErr(cmd.MarkFlagRequired("uuid"))

	return cmd
}
