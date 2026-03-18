package robologger

import (
	"os"
	"runtime"
)

// Global application metadata constants.
const (
	ProgramName string = "Application Name" // Change this to your applications's name.
	Version     string = "0.0.1.dev"        // Change this to your application's version.
	OS          string = runtime.GOOS       //Do not modify this; it is set automatically based on the OS.
)

// Global configuration variables.
// Config globals are in all caps with underscores to distinguish them from other globals.
var (
	//Do not set this variable directly; it is set once based on the user's OS and environment. CONFIG_DIR is the base directory for all application configuration, logs, and data.
	CONFIG_DIR, _ = os.UserConfigDir()

	// Do not set this variable directly; it is set once based on CONFIG_DIR and ProgramName. LOG_DIR is the directory where log files will be stored.
	LOG_DIR = CONFIG_DIR + "/" + ProgramName + "/logs"

	// Do not set this variable directly; it is set once based on CONFIG_DIR and ProgramName. DB_DIR is the directory where database files will be stored.
	DB_DIR = CONFIG_DIR + "/" + ProgramName + "/db"

	// Do not set this variable directly, outside main.go init() function; use command line flags to configure it. LOG_TO_CONSOLE determines whether logs should also be output to the console.
	LOG_TO_CONSOLE bool = false

	// Do not set this variable directly, outside main.go init() function; use command line flags to configure it. RUN_LEVEL controls the verbosity of logging and debug mode. Default is 0 (INFO). -4 is debug mode with debug level logging, 4 is quiet mode with only warnings and errors.
	RUN_LEVEL int = 0
)
