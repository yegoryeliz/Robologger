package robologger

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetLogName(t *testing.T) {
	name := setLogName()
	if name == "" {
		t.Error("setLogName() returned empty string")
	}
	if strings.Contains(name, ":") || strings.Contains(name, "T") || strings.Contains(name, "+") {
		t.Errorf("setLogName() returned string with invalid characters: %s", name)
	}
}

func TestSetGetLogger(t *testing.T) {
	initialLogger := GetLogger()
	if initialLogger == nil {
		t.Error("GetLogger() returned nil initially")
	}

	slogLogger := &Slog{}
	SetLogger(slogLogger)
	if GetLogger() != slogLogger {
		t.Error("GetLogger() did not return the set Slog logger")
	}

	// Setting to nil should reset it to default &LogFile{}
	SetLogger(nil)
	switch GetLogger().(type) {
	case *LogFile:
		// Expected behavior
	default:
		t.Error("SetLogger(nil) did not reset to *LogFile")
	}
}

func TestDebugLogging(t *testing.T) {
	tests := []struct {
		level    int
		expected bool
	}{
		{-4, true},
		{0, false},
		{1, false},
		{-1, false},
	}

	for _, tt := range tests {
		result := debugLogging(tt.level)
		if result != tt.expected {
			t.Errorf("debugLogging(%d) = %v, expected %v", tt.level, result, tt.expected)
		}
	}
}

func TestStartEndLog(t *testing.T) {
	if StartLog() == "" {
		t.Error("StartLog() returned empty string")
	}
	if EndLog() == "" {
		t.Error("EndLog() returned empty string")
	}
}

func TestBuildPairwiseEntry(t *testing.T) {
	expected := "INFO: Test message\n\tKEY1: value1\n\tKEY2: 2\n"
	result := buildPairwiseEntry("INFO", "Test message", "key1", "value1", "key2", 2)
	if result != expected {
		t.Errorf("buildPairwiseEntry() = %q, expected %q", result, expected)
	}

	// Test behavior when an odd number of arguments is provided
	expectedOdd := "ERROR: Msg\n\tKEY: val\n\todd\n"
	resultOdd := buildPairwiseEntry("ERROR", "Msg", "key", "val", "odd")
	if resultOdd != expectedOdd {
		t.Errorf("buildPairwiseEntry() (odd) = %q, expected %q", resultOdd, expectedOdd)
	}
}

func TestBuildHeader(t *testing.T) {
	header := buildHeader()
	if !strings.Contains(header, "-----------------------------------") {
		t.Error("buildHeader() missing header formatting")
	}
	if !strings.Contains(header, "Executed by:") {
		t.Error("buildHeader() missing 'Executed by:'")
	}
}

func TestLogFile_Methods(t *testing.T) {
	// Override CONFIG_DIR to use a temporary directory for this test
	originalConfigDir := CONFIG_DIR
	CONFIG_DIR = t.TempDir()
	defer func() {
		CloseLogFiles()
		CONFIG_DIR = originalConfigDir
	}()

	l := &LogFile{}

	l.Info("test info")
	l.Error("test error")
	l.Warning("test warning")

	// Test Debug with RUN_LEVEL set to Debug mode (-4)
	originalRunLevel := RUN_LEVEL
	RUN_LEVEL = -4
	l.Debug("test debug")

	// Test Debug with RUN_LEVEL set to Info mode (0)
	RUN_LEVEL = 0
	l.Debug("ignored debug")
	RUN_LEVEL = originalRunLevel

	// Ensure the log file is written and read its contents
	logFilePath := filepath.Join(CONFIG_DIR, ProgramName, "logs", logName+".log")
	contentBytes, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "INFO: test info") {
		t.Errorf("Expected info message in log")
	}
	if !strings.Contains(content, "ERROR: test error") {
		t.Errorf("Expected error message in log")
	}
	if !strings.Contains(content, "WARNING: test warning") {
		t.Errorf("Expected warning message in log")
	}
	if !strings.Contains(content, "DEBUG: test debug") {
		t.Errorf("Expected debug message in log")
	}
	if strings.Contains(content, "ignored debug") {
		t.Errorf("Did not expect 'ignored debug' to be logged when RUN_LEVEL != -4")
	}
}

func TestLogFile_Fatal(t *testing.T) {
	// Fatal calls os.Exit(1), which terminates the test runner.
	// To test it, we run this test function in a subprocess.
	if os.Getenv("TEST_LOGFILE_FATAL") == "1" {
		CONFIG_DIR = t.TempDir()
		l := &LogFile{}
		l.Fatal("fatal error message", "key", "value")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLogFile_Fatal")
	cmd.Env = append(os.Environ(), "TEST_LOGFILE_FATAL=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", e.ExitCode())
		}
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestSlog_Methods(t *testing.T) {
	// Override CONFIG_DIR to use a temporary directory for this test
	originalConfigDir := CONFIG_DIR
	CONFIG_DIR = t.TempDir()
	defer func() {
		CloseLogFiles()
		CONFIG_DIR = originalConfigDir
	}()

	l := &Slog{}

	l.Info("test slog info")
	l.Error("test slog error")
	l.Warning("test slog warning")
	l.Debug("test slog debug")

	logFilePath := filepath.Join(CONFIG_DIR, ProgramName, "logs", logName+".json")
	contentBytes, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read slog file: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "test slog info") {
		t.Errorf("Expected info message in slog")
	}
	if !strings.Contains(content, "test slog error") {
		t.Errorf("Expected error message in slog")
	}
	if !strings.Contains(content, "test slog warning") {
		t.Errorf("Expected warning message in slog")
	}
	// Note: We don't check for 'test slog debug' here because the default
	// slog handler level is Info, so it won't be written to the file without changing the logger's level option.
}

func TestSlog_Fatal(t *testing.T) {
	if os.Getenv("TEST_SLOG_FATAL") == "1" {
		CONFIG_DIR = t.TempDir()
		l := &Slog{}
		l.Fatal("fatal error message", "key", "value")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSlog_Fatal")
	cmd.Env = append(os.Environ(), "TEST_SLOG_FATAL=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", e.ExitCode())
		}
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
