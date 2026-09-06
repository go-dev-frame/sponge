package gobash

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	cmds := map[string][]string{
		"pwd":  {},
		"go":   {"env", "GOPATH"},
		"bash": {"-c", "for i in $(seq 1 5); do echo 'test cmd' $i;sleep 0.1; done"},
	}

	for cmd, args := range cmds {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		result := Run(ctx, cmd, args...)
		counter := 0
		for v := range result.StdOut { // Real-time output of logs and error messages
			counter++
			if counter == 1 {
				pid := ParsePid(v)
				t.Logf("pid: %d", pid)
				continue
			}
			t.Log(v)
		}
		if result.Err != nil {
			t.Logf("execute command failed, %v", result.Err)
		}
		fmt.Println()
	}
}

func TestExec(t *testing.T) {
	cmds := map[string][]string{
		"pwd":  {},
		"go":   {"env", "GOROOT"},
		"bash": {"-c", "for i in $(seq 1 5); do echo 'test cmd' $i;sleep 0.1; done"},
	}

	for cmd, args := range cmds {
		out, err := Exec(cmd, args...)
		if err != nil {
			t.Logf("execute command[%s] failed, %v\n", cmd, err)
			continue
		}
		t.Logf("%s\n", out)
	}
}
