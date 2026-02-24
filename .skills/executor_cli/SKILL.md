---
name: "executor_cli"
description: "Паттерн CLI Executor: запуск внешних команд с контекстом, логированием и трейсингом"
modes: [Code, Ask]
---
# Skill: CLI Executor

## Tactical Instructions

```go
// Create CLI executor
cfg := cli.Config{
    Command: "ffmpeg",
}
executor := cli.New(cfg, nil, nil)
defer executor.Close()

// Execute command with arguments
ctx := context.Background()
output, err := executor.Execute(ctx,
    "-i", "input.mp4",
    "-c:v", "libx264",
    "-c:a", "aac",
    "-y", "output.mp4",
)
if err != nil {
    log.Fatal(err)
}
```

### Notes
- Uses `os/exec` standard library internally
- Implements `Executor` interface
- Supports context cancellation
- Logger and tracer are optional second/third arguments to `cli.New(cfg, logger, tracer)`
- `%q` quoting in error messages: `errors.Wrapf(err, "command %q not found", e.cmd)`
