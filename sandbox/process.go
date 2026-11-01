//go:build darwin || linux

package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const killWaitDelay = 2 * time.Second

func Wrap(ctx context.Context, binary string, args []string, workDir string, opt *Option) (*exec.Cmd, error) {
	cmd, err := wrap(ctx, binary, args, workDir, opt)
	if err != nil {
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = killWaitDelay
	return cmd, nil
}

func Run(cmd *exec.Cmd) error {
	err := cmd.Run()
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if errors.Is(err, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() {
		return nil
	}
	return err
}
