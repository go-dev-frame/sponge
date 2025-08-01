## gobash

Execute commands, scripts, executables in the go environment with live log output.

<br>

## Example of use

### Run

Run executes commands and can actively end them, returning logs and error messages in real time, recommended.

```go

    arg := "for i in $(seq 1 5); do echo 'test cmd' $i;sleep 1; done"
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // timeout control
    defer cancel()

    result := Run(ctx, "bash", "-c", arg)
    // real-time output of logs and error messages
    for v := range result.StdOut {
        fmt.Printf(v)
    }
    if result.Err != nil {
        fmt.Println("execute command failed,", result.Err.Error())
    }
```

<br>

### Exec

Exec is suitable for executing a single non-blocking command, outputting standard and error logs, but the log output is not real-time, note: if the execution of the command is permanently blocked, it will cause a concurrent leak

```go
    arg := "for i in $(seq 1 5); do echo 'test cmd' $i;sleep 1; done"
    out, err := Exec("bash", "-c", arg)
    if err != nil {
        return
    }
    fmt.Println(string(out))
```
