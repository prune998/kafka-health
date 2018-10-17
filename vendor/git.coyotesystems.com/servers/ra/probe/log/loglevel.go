package log

import (
	"flag"
	"fmt"

	envflag "github.com/namsral/flag"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level represents the minimum level of logging for a program
type Level int

const (
	// FatalLevel indicates only fatal errors are logged
	FatalLevel Level = iota
	// ErrorLevel indicates only errors are logged
	ErrorLevel
	// WarnLevel indicates both errors and warnings are logged
	WarnLevel
	// InfoLevel indicates errors, warns, and informational messages are logged
	InfoLevel
	// DebugLevel indicates everything is logged
	DebugLevel
)

// String returns a string representation of the LogLevel
func (l Level) String() string {
	switch l {
	case FatalLevel:
		return "fatal"
	case ErrorLevel:
		return "error"
	case WarnLevel:
		return "warn"
	case InfoLevel:
		return "info"
	case DebugLevel:
		return "debug"
	default:
		return fmt.Sprintf("invalid level(%d)", l)
	}
}

// Set sets the log value from its string representation ("error", "warn",
// "info", or "debug".) This is used for command line parsing
func (l *Level) Set(lvl string) error {
	switch lvl {
	case "fatal", "FATAL":
		*l = FatalLevel
	case "error", "ERROR":
		*l = ErrorLevel
	case "warn", "WARN":
		*l = WarnLevel
	case "info", "INFO":
		*l = InfoLevel
	case "debug", "DEBUG":
		*l = DebugLevel
	default:
		return fmt.Errorf("invalid level %s", lvl)
	}
	return nil
}

func tozaplevel(lvl Level) zapcore.Level {
	switch lvl {
	case FatalLevel:
		return zap.FatalLevel
	case ErrorLevel:
		return zap.ErrorLevel
	case WarnLevel:
		return zap.WarnLevel
	case InfoLevel:
		return zap.InfoLevel
	case DebugLevel:
		return zap.DebugLevel
	default:
		return zap.InfoLevel // by default we go with info
	}
}

// LevelVar implements a helper function to parse a log.Level parameter to
// mimick the helpers implemented in the flag package. The first parameter must
// be a pointer to a FlagSet from either the Go standard library flag package or
// from github.com/namsral/flag
func LevelVar(flagSet interface{}, name string, defval Level, usage string) *Level {
	l := new(Level)
	*l = defval
	switch fs := flagSet.(type) {
	case *flag.FlagSet:
		fs.Var(l, name, usage)
	case *envflag.FlagSet:
		fs.Var(l, name, usage)
	default:
		panic(fmt.Sprintf("wrong type %T in flagSet parameter", flagSet))
	}
	return l
}
