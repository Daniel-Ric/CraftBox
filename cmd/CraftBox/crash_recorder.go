package main

import (
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"time"
)

const crashRecorderArgument = "--craftbox-crash-recorder"

func init() {
	if len(os.Args) == 4 && os.Args[1] == crashRecorderArgument {
		err := diagnostics.RecordCrashReady(os.Stdin, os.Args[2], os.Args[3], diagnostics.CrashBytes, func() { fmt.Fprintln(os.Stdout, "ready") })
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
func startCrashRecorder(m *diagnostics.Manager) (func(), error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(executable, crashRecorderArgument, m.Store().Path(), m.ID())
	hideDiagnosticProcess(cmd)
	cmd.Stdin = reader
	readyReader, err := cmd.StdoutPipe()
	if err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	reader.Close()
	ready := make(chan error, 1)
	go func() {
		var ack [6]byte
		_, e := io.ReadFull(readyReader, ack[:])
		if e == nil && string(ack[:]) != "ready\n" {
			e = fmt.Errorf("invalid recorder handshake")
		}
		ready <- e
	}()
	select {
	case err = <-ready:
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("crash recorder did not become ready")
	}
	if err != nil {
		writer.Close()
		cmd.Process.Kill()
		cmd.Wait()
		return nil, err
	}
	if err = debug.SetCrashOutput(writer, debug.CrashOptions{}); err != nil {
		writer.Close()
		cmd.Wait()
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	return func() {
		_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
		writer.Close()
		select {
		case err := <-done:
			if err != nil {
				m.Record("error", "diagnostics", "crash.recorder.failed", err.Error(), nil)
			}
		case <-time.After(3 * time.Second):
			m.Record("warning", "diagnostics", "crash.recorder.pending", "Crash recorder is still flushing", nil)
		}
	}, nil
}
