package exec

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"

	"github.com/creack/pty"
)

type Command struct {
	cmd      *exec.Cmd
	envStore *EnvVarStore
	ptty     *os.File
}

type OnExecEndFn func(exitCode int, errMsg string, a ...any)

func (c *Command) Environ() []string {
	if c.cmd != nil {
		return c.cmd.Environ()
	}
	return nil
}

func (c *Command) String() string {
	if c.cmd != nil {
		return c.cmd.String()
	}
	return ""
}

func (c *Command) MainCmd() string {
	if len(c.cmd.Args) > 0 {
		return c.cmd.Args[0]
	}
	return ""
}

func (c *Command) Pid() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return -1
}

// OnPreExec execute all pre exec env functions
func (c *Command) OnPreExec() error {
	for _, env := range c.envStore.store {
		if env.OnPreExec == nil {
			continue
		}
		if err := env.OnPreExec(); err != nil {
			return fmt.Errorf("failed storing environment variable %q, err=%v", env.Key, err)
		}
	}
	return nil
}

// OnPostExec execute all post exec env functions
func (c *Command) OnPostExec() error {
	for _, env := range c.envStore.store {
		if env.OnPostExec == nil {
			continue
		}
		if err := env.OnPostExec(); err != nil {
			return fmt.Errorf("failed storing environment variable %q, err=%v", env.Key, err)
		}
	}
	return nil
}

func (c *Command) Run(streamWriter io.WriteCloser, stdinInput []byte, onExecEnd OnExecEndFn, clientArgs ...string) error {
	pipeStdout, err := c.cmd.StdoutPipe()
	if err != nil {
		onExecEnd(InternalErrorExitCode, "internal error, failed returning stdout pipe")
		return err
	}
	pipeStderr, err := c.cmd.StderrPipe()
	if err != nil {
		onExecEnd(InternalErrorExitCode, "internal error, failed returning stderr pipe")
		return err
	}
	if err := c.OnPreExec(); err != nil {
		onExecEnd(InternalErrorExitCode, "internal error, failed executing pre command")
		return fmt.Errorf("failed executing pre command, err=%v", err)
	}
	var stdin bytes.Buffer
	c.cmd.Stdin = &stdin
	exitCode := InternalErrorExitCode
	if err := c.cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			// path not found error exit code
			exitCode = 127
		}
		onExecEnd(exitCode, "failed starting command")
		return err
	}
	if _, err := stdin.Write(stdinInput); err != nil {
		onExecEnd(InternalErrorExitCode, "internal error, failed writing input")
		return err
	}
	copyBuffer(pipeStdout, streamWriter, 1024)
	copyBuffer(pipeStderr, streamWriter, 1024)

	go func() {
		exitCode = 0
		err := c.cmd.Wait()
		if err != nil {
			if exErr, ok := err.(*exec.ExitError); ok {
				exitCode = exErr.ExitCode()
				if exitCode == -1 {
					exitCode = InternalErrorExitCode
				}
			}
		}
		if err := c.OnPostExec(); err != nil {
			fmt.Printf("failed executing post command, err=%v", err)
		}
		onExecEnd(exitCode, "failed executing command, err=%v", err)
	}()
	return nil
}

func (c *Command) RunOnTTY(stdoutWriter io.WriteCloser, onExecEnd OnExecEndFn, clientArgs ...string) error {
	// Start the command with a pty.
	if err := c.OnPreExec(); err != nil {
		return fmt.Errorf("failed executing pre execution command, err=%v", err)
	}

	ptmx, err := pty.Start(c.cmd)
	if err != nil {
		return fmt.Errorf("failed starting pty, err=%v", err)
	}
	c.ptty = ptmx
	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- SIGWINCH                                // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	go func() {
		exitCode := 0
		defer func() {
			log.Printf("exit-code=%v - closing tty ...", exitCode)
			if err := ptmx.Close(); err != nil {
				log.Printf("failed closing tty, err=%v", err)
			}
			// onExecEnd(exitCode, "failed closing tty, err=%v", err)
			if err := c.OnPostExec(); err != nil {
				// TODO: warn
				log.Printf("failed executing post execution command, err=%v", err)
			}
		}()
		// TODO: need to make distinction between stderr / stdout when writing back to client
		if _, err := io.Copy(stdoutWriter, ptmx); err != nil {
			log.Printf("failed copying stdout from tty, err=%v", err)
		}

		err := c.cmd.Wait()
		if err != nil {
			if exErr, ok := err.(*exec.ExitError); ok {
				exitCode = exErr.ExitCode()
				// assume that it was killed or interrupted
				// because the process is probably started already
				if exitCode == -1 {
					exitCode = 1
				}
			}
		}
		onExecEnd(exitCode, "failed executing command, err=%v", err)
	}()

	return nil
}

func (c *Command) WriteTTY(data []byte) (int, error) {
	if c.ptty != nil {
		return c.ptty.Write(data)
	}
	return 0, nil
}

func NewCommand(rawEnvVarList map[string]interface{}, args ...string) (*Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("connection must be at least one argument")
	}
	envStore, err := newEnvVarStore(rawEnvVarList)
	if err != nil {
		return nil, err
	}
	mainCmd := args[0]
	execArgs := args[1:]
	if len(execArgs) > 0 {
		var err error
		execArgs, err = expandEnvVarToCmd(envStore, execArgs)
		if err != nil {
			return nil, err
		}
	}
	c := &Command{envStore: envStore}
	c.cmd = exec.Command(mainCmd, execArgs...)
	c.cmd.Env = envStore.ParseToKeyVal()
	c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("PATH=%v", os.Getenv("PATH")))
	return c, nil
}

func copyBuffer(reader io.ReadCloser, w io.Writer, bufSize int) {
	r := bufio.NewReader(reader)
	buf := make([]byte, bufSize)
	go func() {
		for {
			n, err := r.Read(buf[:])
			if n > 0 {
				if _, err := w.Write(buf[:n]); err != nil {
					panic(err)
				}
				continue
			}
			if err != nil {
				if err == io.EOF {
					break
				}
			}
		}
	}()
}