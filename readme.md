
# RoboLogger

## Introduction

RoboLogger (Robust Logger) is an opinionated Go logging package that writes semi-structured text logs.
It supports two logger implementations behind one interface as well as console output:

- `LogFile`: opinionated,human-readable semi-structured text logs
- `Slog`: JSON logs using Go's `log/slog` for when machine parsable logs are prefered.

Additionally it provides a `FATAL` level that Go's Slog does not implement.

## What RoboLogger Does

RoboLogger gives you a simple global logger (`Log`) with methods:

- `INFO`
- `WARNING`
- `ERROR`
- `DEBUG`
- `FATAL`

By default, logs go to file only. If `LOG_TO_CONSOLE` is true, logs are sent to both file and stdout.

## Log File Location

The default base directory is `os.UserConfigDir()` and paths are built from the `ProgramName` constant.

- Text logger file: `<CONFIG_DIR>/<ProgramName>/logs/<timestamp>.log`
- JSON logger file: `<CONFIG_DIR>/<ProgramName>/logs/<timestamp>.json`

`timestamp` is based on RFC3339 time with safe filename formatting.

## Logger Modes

### LogFile (default)

Human-readable output with pairwise key-value formatting:

```text
2016/03/17 16:59:00 INFO: user signed in
    USER_ID: 123
    METHOD: oauth
```

### Slog

Structured JSON logging through `log/slog`. Includes a custom fatal level constant:

- `LevelFatal = slog.Level(9)`

`Fatal` logs at this level and then terminates the process with exit code `1`.

## Run Level Behavior

`RUN_LEVEL` controls debug logging behavior for `LogFile`:

- `-4`: debug mode enabled (`Debug` entries are written)
- any other value: debug entries are ignored

The package also provides three helpers:

- `StartLog()` returns a start-of-session banner string
- `EndLog()` returns an end-of-session banner string

These strings change wording when `RUN_LEVEL == -4`.

## Global Configuration Variables

Defined in `RoboLoggerConfig.go`:

- `ProgramName` (default placeholder: `Application Name`)
- `Version` (default: `0.0.1.dev`)
- `OS` (auto-detected)
- `CONFIG_DIR` (from `os.UserConfigDir()`)
- `LOG_DIR` (derived paths)
- `LOG_TO_CONSOLE` (default `false`)
- `RUN_LEVEL` (default `0`)

## Typical Usage

```go
package main

import (
    "flag"
    "fmt"
 
    robo "github.com/yegoryeliz/robo"
)


func init() {
    // Optional runtime config
    // Flag switch
    dFlag := flag.Bool("d", false, "Enable quiet debug logging")
    vFlag := flag.Bool("v", false, "Enable verbose debug console logging")
    qFlag := flag.Bool("q", false, "Enable quiet logging")
    flag.Parse()

    robo.RUN_LEVEL = func() int {
        switch {
        case *dFlag:
            slog.SetLogLoggerLevel(slog.LevelDebug)
            return -4
        case *vFlag:
            config.LOG_TO_CONSOLE = true
            slog.SetLogLoggerLevel(slog.LevelDebug)
            return -4
        case *qFlag:
            slog.SetLogLoggerLevel(slog.LevelWarn)
            return 4

        default:
            slog.SetLogLoggerLevel(slog.LevelInfo)
            return 0
        }
    }()

func main() {
    robo.SetLogger(&robo.LogFile{})

    robo.Log.Info(robo.StartLog()) 
    defer func() {
        robo.Log.Info(robo.EndLog()) 
        robo.CloseLogFiles()         
    }()

    robo.Log.Info("service started", "port", 8080)
    robo.Log.Debug("request details", "trace_id", "abc123")
    robo.Log.Info(robo.EndLog())

    robo.SetLogger(&robo.Slog{})
    robo.Log.Info("switched logger", "mode", "json")

}
```

### Outputs

Log file

```log
-----------------------------------
Application Name v0.0.1 for linux 
Executed by: root
Hostname: toolbox
Time: 2026-03-18T16:05:15+01:00
-----------------------------------

2026/03/18 16:05:15 INFO: ###### START OF DEBUG SESSION ######
2026/03/18 16:05:15 INFO: service started
        PORT: 8080
2026/03/18 16:05:15 DEBUG: request details
        TRACE_ID: abc123
2026/03/18 16:05:15 INFO: ###### END OF DEBUG SESSION ######

```

JSON Slog

```JSON
{"time":"2026-03-18T16:05:15.810208463+01:00","level":"INFO","msg":"switched logger","mode":"json"}
```

## Notes

- `Fatal` always exits the process with code `1` after writing fatal messages.
