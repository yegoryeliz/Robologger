package robologger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LogFile struct {
	Logger *log.Logger
}

type Slog struct {
	Logger *slog.Logger
}

const LevelFatal = slog.Level(9)

var (
	logFileHandle  *os.File
	slogFileHandle *os.File
	logOnce        sync.Once
	slogOnce       sync.Once
	logName        string = setLogName()
	hostname, _           = os.Hostname()
	username, _           = user.Current()
	Log            Logger = &LogFile{}
)

type Logger interface {
	Error(string, ...any)
	Info(string, ...any)
	Debug(string, ...any)
	Warning(string, ...any)
	Fatal(string, ...any)
}

func setLogName() (logTime string) {
	l, _, _ := strings.Cut(strings.ReplaceAll(strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", ""), "T", "_"), "+")
	return l
}

func SetLogger(l Logger) {
	if l == nil {
		Log = &LogFile{}
		return
	}
	Log = l
}

func GetLogger() Logger {
	return Log
}

func debugLogging(level int) bool {
	switch level {
	case -4: //Debug mode
		return true
	default:
		return false
	}
}

// ErrLog is a compatibility wrapper for older call sites.
func ErrLog(level string, args ...any) {
	if len(args) == 0 {
		Log.Info(level)
		return
	}

	msg := fmt.Sprint(args...)
	lvl := strings.ToLower(level)

	switch {
	case strings.Contains(lvl, "error"):
		Log.Error(msg)
	case strings.Contains(lvl, "warning"):
		Log.Warning(msg)
	case strings.Contains(lvl, "debug"):
		if RUN_LEVEL == -4 {
			Log.Debug(msg)
		}
	default:
		Log.Info(msg)
	}
}

func buildHeader() string {
	header := "-----------------------------------\n"
	header += fmt.Sprintf("%s v%s for %s \n", ProgramName, Version, OS)
	header += fmt.Sprintf("Executed by: %s\n", username.Username)
	header += fmt.Sprintf("User: %s\n", hostname)
	header += fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC3339))
	header += "-----------------------------------\n\n"
	return header
}

func StartLog() (msg string) {
	// Start of program logging
	if RUN_LEVEL == -4 { //Debug mode
		msg = "###### START OF DEBUG SESSION ######"
	} else {
		msg = "****** START OF SESSION ******"
	}
	return msg
}

func EndLog() (msg string) {
	// End of program logging
	if RUN_LEVEL == -4 { //Debug mode
		msg = "###### END OF DEBUG SESSION ######"
	} else {
		msg = "****** END OF SESSION ******"
	}
	return msg
}

func buildPairwiseEntry(level string, msg string, args ...any) string {
	entry := fmt.Sprintf("%s: %s\n", level, msg)

	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			entry += fmt.Sprintf("\t%s: %v\n", strings.ToUpper(fmt.Sprint(args[i])), args[i+1])
			continue
		}
		entry += fmt.Sprintf("\t%v\n", args[i])
	}

	return entry
}

func (l *LogFile) init() {
	logOnce.Do(func() {
		logFilePath := filepath.Join(CONFIG_DIR, ProgramName, "logs", logName+".log")
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
			log.Printf("failed to create log directory: %v", err)
			return
		}
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Printf("failed to open log file: %v", err)
			return
		}
		logFileHandle = file

		if LOG_TO_CONSOLE {
			l.Logger = log.New(io.MultiWriter(file, os.Stdout), "", log.LstdFlags)
		} else {
			l.Logger = log.New(file, "", log.LstdFlags)
		}

		flags := l.Logger.Flags()
		l.Logger.SetFlags(0)
		l.Logger.Print(buildHeader())
		l.Logger.SetFlags(flags)
	})
}

func (l *Slog) init() {
	slogOnce.Do(func() {
		logFilePath := filepath.Join(CONFIG_DIR, ProgramName, "logs", logName+".json")
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
			log.Printf("failed to create slog directory: %v", err)
			return
		}
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Printf("failed to open slog file: %v", err)
			return
		}
		slogFileHandle = file

		handlerOpts := &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					if level, ok := a.Value.Any().(slog.Level); ok && level == LevelFatal {
						a.Value = slog.StringValue("FATAL")
					}
				}
				return a
			},
		}

		if LOG_TO_CONSOLE {
			l.Logger = slog.New(slog.NewJSONHandler(io.MultiWriter(file, os.Stdout), handlerOpts))
		} else {
			l.Logger = slog.New(slog.NewJSONHandler(file, handlerOpts))
		}
	})
}

// CloseLogFiles cleanly closes the persistent log file descriptors
func CloseLogFiles() {
	if logFileHandle != nil {
		logFileHandle.Close()
	}
	if slogFileHandle != nil {
		slogFileHandle.Close()
	}
}

// Slog methods
func (l *Slog) Error(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Error(msg, args...)
	}
}

func (l *Slog) Info(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Info(strings.ReplaceAll(msg, "\n", ""), args...)
	}
}

func (l *Slog) Debug(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Debug(msg, args...)
	}
}

func (l *Slog) Warning(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Warn(msg, args...)
	}
}

func (l *Slog) Fatal(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Log(context.Background(), LevelFatal, msg, args...)
		l.Logger.Log(context.Background(), LevelFatal, "****** FATAL ERROR, SESSION ENDED ******")
	}
	CloseLogFiles()
	os.Exit(1)
}

// LogFile methods
func (l *LogFile) Error(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("ERROR", msg, args...))
	}
}

func (l *LogFile) Info(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("INFO", msg, args...))
	}
}

func (l *LogFile) Debug(msg string, args ...any) {
	l.init()
	if l.Logger != nil && debugLogging(RUN_LEVEL) {
		l.Logger.Print(buildPairwiseEntry("DEBUG", msg, args...))
	}
}

func (l *LogFile) Warning(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("WARNING", msg, args...))
	}
}

func (l *LogFile) Fatal(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("FATAL", msg, args...))
		l.Logger.Print(buildPairwiseEntry("INFO", "****** FATAL ERROR, SESSION ENDED ****** \n"))
	}
	CloseLogFiles()
	os.Exit(1)
}
