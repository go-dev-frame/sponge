//go:build windows
// +build windows

package process

import (
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/windows" // <--- 修改：导入 "golang.org/x/sys/windows"
)

// Platform-specific implementation for starting a test process.
func getTestProcessCmd2(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	// Create a new console for the process, so we can send Ctrl-Break signals to it
	// without affecting the current test runner console.
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_CONSOLE,
	}
	return cmd
}

func TestWindows_IsProcessRunning_True(t *testing.T) {
	cmd := getTestProcessCmd2("timeout", "/t", "10")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid
	if !isProcessRunning(pid) {
		t.Errorf("isProcessRunning for PID %d should be true", pid)
	}
}

func TestWindows_TryGracefulExit(t *testing.T) {
	cmd := getTestProcessCmd2("timeout", "/t", "10")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	pid := cmd.Process.Pid
	t.Log("Started process with PID", pid)

	// Wait a bit to ensure the process and its console are fully initialized
	time.Sleep(time.Second)

	ok := tryGracefulExit(pid)
	if !ok {
		err = cmd.Process.Kill()
		if err != nil {
			t.Log("tryGracefulExit failed, but it should have succeeded")
			return
		}
	}

	// Wait for the process to exit
	cmd.Wait()

	if isProcessRunning(pid) {
		t.Errorf("Process %d should not be running after graceful exit", pid)
	}
}

// Test case where graceful exit fails because the process has no console.
func TestWindows_TryGracefulExit_NoConsole(t *testing.T) {
	// Start a process without a new console. `ping` is a good candidate.
	cmd := exec.Command("ping", "-n", "10", "127.0.0.1")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	defer cmd.Process.Kill()

	pid := cmd.Process.Pid

	// This should fail because AttachConsole won't work on a process without its own console.
	ok := tryGracefulExit(pid)
	if ok {
		t.Error("tryGracefulExit should have failed for a process with no console, but it succeeded")
	}
}

func TestWindows_ForceKill(t *testing.T) {
	cmd := getTestProcessCmd2("timeout", "/t", "10")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start process: %v", err)
	}
	pid := cmd.Process.Pid

	ok := forceKill(pid)
	if !ok {
		cmd.Process.Kill()
		t.Fatal("forceKill failed, but it should have succeeded")
	}

	time.Sleep(100 * time.Millisecond) // Give OS time to update process table

	if isProcessRunning(pid) {
		t.Errorf("Process %d should not be running after force kill", pid)
	}
}
