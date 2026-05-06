package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const watchInterval = 800 * time.Millisecond

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var current *exec.Cmd
	signature := ""

	for {
		nextSignature, err := sourceSignature(".")
		if err != nil {
			log.Printf("scan source: %v", err)
		}
		if current == nil || nextSignature != signature {
			if current != nil {
				stopCommand(current)
			}
			signature = nextSignature
			current = startServer(ctx)
		}

		select {
		case <-ctx.Done():
			if current != nil {
				stopCommand(current)
			}
			return
		case <-time.After(watchInterval):
		}
	}
}

func startServer(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/api")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		log.Printf("start server: %v", err)
		return nil
	}
	log.Printf("server started with pid %d", cmd.Process.Pid)
	return cmd
}

func stopCommand(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	log.Printf("server restarting")
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		err = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		err = cmd.Process.Signal(syscall.SIGTERM)
	}
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		if pgid > 0 {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
	}
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		if pgid > 0 {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
	}
}

func sourceSignature(root string) (string, error) {
	latest := time.Time{}
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "tmp", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		count++
		return nil
	})
	if err != nil {
		return "", err
	}
	return latest.Format(time.RFC3339Nano) + ":" + strconv.Itoa(count), nil
}
