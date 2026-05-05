package gmcore_process

// Package gmcore_process provides process spawning, management, and streaming utilities.
//
// Examples:
//
//	// Simple run and wait
//	result, err := Run("echo", []string{"hello"}, nil)
//
//	// Spawn and manage
//	proc, _ := Spawn("sleep", []string{"10"}, nil)
//	go func() { time.Sleep(1*time.Second); proc.Kill() }()
//	exitCode, _ := proc.Wait()
//
//	// Streaming output
//	RunStreaming("go", []string{"build"}, nil,
//	    func(line string) { fmt.Println("STDOUT:", line) },
//	    func(line string) { fmt.Println("STDERR:", line) })

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Process struct {
	cmd    *exec.Cmd
	done   bool
	lock   sync.Mutex
}

type Options struct {
	Dir       string
	Env       []string
	UID       uint32
	GID       uint32
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	ExtraFiles []*os.File
}

type Result struct {
	Pid       int
	ExitCode  int
	Duration  time.Duration
	Stdout    string
	Stderr    string
}

func Spawn(name string, args []string, opts *Options) (*Process, error) {
	cmd := exec.Command(name, args...)

	if opts != nil {
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if opts.Env != nil {
			cmd.Env = opts.Env
		}
		if opts.Dir == "" && opts.Env == nil {
			cmd.Env = os.Environ()
		}
		if opts.Stdin != nil {
			cmd.Stdin = opts.Stdin
		}
		if opts.Stdout != nil {
			cmd.Stdout = opts.Stdout
		}
		if opts.Stderr != nil {
			cmd.Stderr = opts.Stderr
		}
		if opts.ExtraFiles != nil {
			cmd.ExtraFiles = opts.ExtraFiles
		}
	} else {
		cmd.Env = os.Environ()
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	return &Process{
		cmd:  cmd,
		done: false,
	}, nil
}

func (p *Process) Pid() int {
	return p.cmd.Process.Pid
}

func (p *Process) Wait() (int, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.done {
		return p.cmd.ProcessState.ExitCode(), nil
	}

	err := p.cmd.Wait()
	p.done = true

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return ws.ExitStatus(), nil
			}
		}
		return -1, fmt.Errorf("process wait failed: %w", err)
	}

	if ws, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		return ws.ExitStatus(), nil
	}

	return 0, nil
}

func (p *Process) WaitWithTimeout(timeout time.Duration) (int, bool, error) {
	done := make(chan struct{})
	var exitCode int
	var err error

	go func() {
		exitCode, err = p.Wait()
		close(done)
	}()

	select {
	case <-done:
		return exitCode, true, err
	case <-time.After(timeout):
		return -1, false, nil
	}
}

func (p *Process) Signal(sig syscall.Signal) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	return p.cmd.Process.Signal(sig)
}

func (p *Process) Kill() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill failed: %w", err)
	}

	p.done = true
	return nil
}

func (p *Process) Terminate() error {
	return p.Signal(syscall.SIGTERM)
}

func (p *Process) IsDone() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.done
}

func (p *Process) Running() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.done || p.cmd.Process == nil {
		return false
	}

	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

func Run(name string, args []string, opts *Options) (*Result, error) {
	cmd := exec.Command(name, args...)

	if opts != nil {
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if opts.Env != nil {
			cmd.Env = opts.Env
		}
		if opts.Dir == "" && opts.Env == nil {
			cmd.Env = os.Environ()
		}
	} else {
		cmd.Env = os.Environ()
	}

	var stdout, stderr bytes.Buffer
	if opts == nil || opts.Stdout == nil {
		cmd.Stdout = &stdout
	}
	if opts == nil || opts.Stderr == nil {
		cmd.Stderr = &stderr
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return &Result{
					ExitCode: ws.ExitStatus(),
					Duration: duration,
					Stdout:   stdout.String(),
					Stderr:   stderr.String(),
				}, nil
			}
		}
	}

	exitCode := 0
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		exitCode = ws.ExitStatus()
	}

	return &Result{
		Pid:       cmd.Process.Pid,
		ExitCode:  exitCode,
		Duration:  duration,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
	}, nil
}

func RunWithContext(ctx context.Context, name string, args []string, opts *Options) (*Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	if opts != nil {
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if opts.Env != nil {
			cmd.Env = opts.Env
		}
		if opts.Dir == "" && opts.Env == nil {
			cmd.Env = os.Environ()
		}
	} else {
		cmd.Env = os.Environ()
	}

	var stdout, stderr bytes.Buffer
	if opts == nil || opts.Stdout == nil {
		cmd.Stdout = &stdout
	}
	if opts == nil || opts.Stderr == nil {
		cmd.Stderr = &stderr
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return &Result{
					ExitCode: ws.ExitStatus(),
					Duration: duration,
					Stdout:   stdout.String(),
					Stderr:   stderr.String(),
				}, nil
			}
		}
	}

	exitCode := 0
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		exitCode = ws.ExitStatus()
	}

	return &Result{
		Pid:       cmd.Process.Pid,
		ExitCode:  exitCode,
		Duration:  duration,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
	}, nil
}

func RunStreaming(name string, args []string, opts *Options, stdoutHandler func(string), stderrHandler func(string)) (*Result, error) {
	cmd := exec.Command(name, args...)

	if opts != nil {
		if opts.Dir != "" {
			cmd.Dir = opts.Dir
		}
		if opts.Env != nil {
			cmd.Env = opts.Env
		}
		if opts.Dir == "" && opts.Env == nil {
			cmd.Env = os.Environ()
		}
	} else {
		cmd.Env = os.Environ()
	}

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	var wg sync.WaitGroup
	var stdout, stderr bytes.Buffer

	if stdoutHandler != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdoutR)
			for scanner.Scan() {
				line := scanner.Text()
				stdout.WriteString(line + "\n")
				stdoutHandler(line)
			}
		}()
	}

	if stderrHandler != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderrR)
			for scanner.Scan() {
				line := scanner.Text()
				stderr.WriteString(line + "\n")
				stderrHandler(line)
			}
		}()
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	stdoutW.Close()
	stderrW.Close()
	wg.Wait()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return &Result{
					ExitCode: ws.ExitStatus(),
					Duration: duration,
					Stdout:   stdout.String(),
					Stderr:   stderr.String(),
				}, nil
			}
		}
	}

	exitCode := 0
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		exitCode = ws.ExitStatus()
	}

	return &Result{
		Pid:       cmd.Process.Pid,
		ExitCode:  exitCode,
		Duration:  duration,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
	}, nil
}

func LookPath(file string) (string, error) {
	return exec.LookPath(file)
}
