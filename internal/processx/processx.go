package processx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func Run(command []string, env []string, verbose bool) (int, error) {
	if len(command) == 0 {
		return 2, errors.New("empty command")
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start command: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[profy] child pid: %d\n", cmd.Process.Pid)
	}

	sigCh := make(chan os.Signal, 8)
	done := make(chan struct{})
	notifySignals(sigCh)
	go func() {
		defer close(done)
		for sig := range sigCh {
			forwardSignal(cmd, sig)
		}
	}()

	err := cmd.Wait()
	signal.Stop(sigCh)
	close(sigCh)
	<-done
	return resolveWaitResult(err)
}

func RunWithReload(command []string, envBuilder func() ([]string, error), reload <-chan struct{}, verbose bool) (int, error) {
	if len(command) == 0 {
		return 2, errors.New("empty command")
	}

	sigCh := make(chan os.Signal, 8)
	notifySignals(sigCh)
	defer signal.Stop(sigCh)

	for {
		env, err := envBuilder()
		if err != nil {
			return 1, err
		}

		cmd := exec.Command(command[0], command[1:]...)
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if runtime.GOOS != "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		}
		if err := cmd.Start(); err != nil {
			return 1, fmt.Errorf("start command: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "[profy] child pid: %d\n", cmd.Process.Pid)
		}

		waitCh := make(chan error, 1)
		go func() { waitCh <- cmd.Wait() }()

		restart := false
		for !restart {
			select {
			case sig := <-sigCh:
				forwardSignal(cmd, sig)
			case <-reload:
				if verbose {
					fmt.Fprintln(os.Stderr, "[profy] env change detected, restarting child process")
				}
				stopCommand(cmd)
				<-waitCh
				restart = true
			case err := <-waitCh:
				return resolveWaitResult(err)
			}
		}
	}
}

func stopCommand(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Signal(os.Interrupt)
		go func() {
			time.Sleep(1500 * time.Millisecond)
			_ = cmd.Process.Kill()
		}()
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	go func(pid int) {
		time.Sleep(2 * time.Second)
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}(cmd.Process.Pid)
}

func resolveWaitResult(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Exited() {
				return status.ExitStatus(), nil
			}
			if status.Signaled() {
				return 128 + int(status.Signal()), nil
			}
		}
		return 1, nil
	}
	return 1, err
}

func notifySignals(sigCh chan<- os.Signal) {
	if runtime.GOOS == "windows" {
		signal.Notify(sigCh, os.Interrupt)
		return
	}
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
}

func forwardSignal(cmd *exec.Cmd, sig os.Signal) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Signal(sig)
		return
	}
	s, ok := sig.(syscall.Signal)
	if !ok {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, s)
}
