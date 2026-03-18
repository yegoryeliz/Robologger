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

// LogFile and Slog are the two robologger implementations. LogFile is an opinionated, human readable, semi-structured file-based logger, while Slog uses the structured logging capabilities of slog with an additional logging level of FATAL. Both implement the Logger interface, which defines the standard logging methods (Error, Info, Debug, Warning, Fatal). The global Log variable is initialized to a default LogFile logger, but can be switched to Slog or any other custom logger that implements the Logger interface using SetLogger().

type LogFile struct {
	Logger *log.Logger
}

type Slog struct {
	Logger *slog.Logger
}

const LevelFatal = slog.Level(9)

var (
	logFileHandle  *os.File                   // used for cleanly closing the log file when the application exits
	slogFileHandle *os.File                   // used for cleanly closing the slog file when the application exits
	logOnce        sync.Once                  // ensures that log file initialization happens only once
	slogOnce       sync.Once                  // ensures that slog file initialization happens only once
	logName        string    = setLogName()   // logName is set once at startup based on the current timestamp, ensuring that each session gets a unique log file.
	hostname, _              = os.Hostname()  // hostname is set once at startup and used in log headers to identify the machine where the logs were generated.
	username, _              = user.Current() // username is set once at startup and used in log headers to identify the user who executed the program.
	Log            Logger    = &LogFile{}     // Log is the global logger instance that can be switched to Slog using SetLogger().
)

type Logger interface {
	Error(string, ...any)   // Error logs an error message with optional key-value pairs for additional context.
	Info(string, ...any)    // Info logs an informational message with optional key-value pairs for additional context.
	Debug(string, ...any)   // Debug logs a debug message with optional key-value pairs for additional context. To emulate log/slog's approach to runlevel filtering, debug messages are only logged if RUN_LEVEL is set to -4 (debug mode).
	Warning(string, ...any) // Warning logs a warning message with optional key-value pairs for additional context.
	Fatal(string, ...any)   // Fatal logs a fatal error message with optional key-value pairs for additional context, then cleanly closes log files and exits the program with status code 1.
}

// setLogName generates a log name based on the current timestamp, formatted as "YYYY-MM-DD_HHMMSS". This ensures that each log file has a unique name based on when the program was executed.
func setLogName() (logTime string) {
	l, _, _ := strings.Cut(strings.ReplaceAll(strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", ""), "T", "_"), "+")
	return l
}

// SetLogger allows users to switch the global Log variable to a different logger implementation that satisfies the Logger interface. If nil is passed, it defaults back to the LogFile implementation.
func SetLogger(l Logger) {
	if l == nil {
		Log = &LogFile{}
		return
	}
	Log = l
}

// GetLogger returns the current global logger instance, allowing other parts of the application to access it without directly referencing the Log variable. This promotes better encapsulation and allows for more flexible logging configurations.
func GetLogger() Logger {
	return Log
}

// debugLogging is a helper function that checks if the current RUN_LEVEL is set to -4 (debug mode). If it is, it returns true, indicating that debug messages should be logged. For any other RUN_LEVEL value, it returns false, meaning that debug messages will be ignored. This allows for easy control over the verbosity of logging based on the configured run level. This function relies on flag parsing in main.go to set the RUN_LEVEL variable (not implemented here). By default, RUN_LEVEL is set to 0 (INFO), so debug messages will not be logged unless RUN_LEVEL is explicitly set to -4.
func debugLogging(level int) bool {
	switch level {
	case -4: //Debug mode
		return true
	default:
		return false
	}
}

// DEPRECATED: ErrLog is a compatibility wrapper for older calls used in the original project that spawned this logger library.
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

// buildHeader constructs a standardized log header that includes the program name, version, operating system, executing user, hostname, and timestamp. This header is included at the beginning of each log file to provide context about the environment in which the logs were generated. The header is formatted with separators for readability and uses the global constants and variables defined in roboConfig.go to populate the relevant information. NOTE: this function is not implemented for Slog.
func buildHeader() string {
	header := "-----------------------------------\n"
	header += fmt.Sprintf("%s v%s for %s \n", ProgramName, Version, OS)
	header += fmt.Sprintf("Executed by: %s\n", username.Username)
	header += fmt.Sprintf("Hostname: %s\n", hostname)
	header += fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC3339))
	header += "-----------------------------------\n\n"
	return header
}

// StartLog generates a standardized start of session string based on the current RUN_LEVEL. Must be called from main.go
func StartLog() (msg string) {
	// Start of program logging
	if RUN_LEVEL == -4 { //Debug mode
		msg = "###### START OF DEBUG SESSION ######"
	} else {
		msg = "****** START OF SESSION ******"
	}
	return msg
}

// EndLog generates a standardized end of session string based on the current RUN_LEVEL. Must be called from main.go
func EndLog() (msg string) {
	// End of program logging
	if RUN_LEVEL == -4 { //Debug mode
		msg = "###### END OF DEBUG SESSION ######"
	} else {
		msg = "****** END OF SESSION ******"
	}
	return msg
}

// buildPairwiseEntry constructs a log entry string in a human-readable format, starting with the log level and message, followed by optional key-value pairs for additional context. The key-value pairs are formatted with the keys in uppercase. If an odd number of arguments is provided, the last argument will be logged without a key. NOTE: this function is not implemented for Slog.
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

// init initializes the LogFile logger by creating the log file if it doesn't exist, setting up the logger to write to the file (and optionally to the console), and writing a standardized header to the log file. The initialization is done using sync.Once to ensure that it only happens once, even if multiple log messages are generated concurrently at startup. If LOG_TO_CONSOLE is true, logs will be written to both the file and standard output; otherwise, they will only be written to the file.
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

// init initializes the Slog logger by creating the log file if it doesn't exist, setting up the slog logger to write to the file (and optionally to the console), and configuring a custom handler option to a "FATAL" key for fatal log entries. If LOG_TO_CONSOLE is true, logs will be written to both the file and standard output; otherwise, they will only be written to the file. NOTE: this function does not write a header to the slog file, as slog's structured logging format is designed for machine parsing rather than human readability.
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

// Error logs an error message with optional key-value pairs for additional context.
func (l *Slog) Error(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Error(msg, args...)
	}
}

// Info logs an informational message with optional key-value pairs for additional context.
func (l *Slog) Info(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Info(strings.ReplaceAll(msg, "\n", ""), args...)
	}
}

// Debug logs a debug message with optional key-value pairs for additional context.
func (l *Slog) Debug(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Debug(msg, args...)
	}
}

// Warning logs a warning message with optional key-value pairs for additional context.
func (l *Slog) Warning(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Warn(msg, args...)
	}
}

// Fatal logs a fatal error message with optional key-value pairs for additional context, then cleanly closes log files and exits the program with status code 1.
func (l *Slog) Fatal(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Log(context.Background(), LevelFatal, msg, args...)
	}
	CloseLogFiles()
	os.Exit(1)
}

// Error logs an error message with optional key-value pairs for additional context.
func (l *LogFile) Error(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("ERROR", msg, args...))
	}
}

// Info logs an informational message with optional key-value pairs for additional context.
func (l *LogFile) Info(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("INFO", msg, args...))
	}
}

// Debug logs a debug message with optional key-value pairs for additional context. To emulate log/slog's approach to runlevel filtering, debug messages are only logged if RUN_LEVEL is set to -4 (debug mode).
func (l *LogFile) Debug(msg string, args ...any) {
	l.init()
	if l.Logger != nil && debugLogging(RUN_LEVEL) {
		l.Logger.Print(buildPairwiseEntry("DEBUG", msg, args...))
	}
}

// Warning logs a warning message with optional key-value pairs for additional context.
func (l *LogFile) Warning(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("WARNING", msg, args...))
	}
}

// Fatal logs a fatal error message with optional key-value pairs for additional context, then cleanly closes log files and exits the program with status code 1.
func (l *LogFile) Fatal(msg string, args ...any) {
	l.init()
	if l.Logger != nil {
		l.Logger.Print(buildPairwiseEntry("FATAL", msg, args...))
		l.Logger.Print(buildPairwiseEntry("INFO", "****** FATAL ERROR, SESSION ENDED ****** \n"))
	}
	CloseLogFiles()
	os.Exit(1)
}
