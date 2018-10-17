package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalMu sync.RWMutex // protects the global logger
	globalL  *Logger      // global logger
)

func init() {
	// by default, the global logger is set to Info
	globalL = New(WithInfo())
}

// Logger allows for structured logging
type Logger struct {
	mu       sync.Mutex       // ensures atomic writes
	lvl      Level            // the Log Level
	l        *zap.Logger      // a zap logger
	with     []zap.Field      // list of default fields
	alvl     *zap.AtomicLevel // the zap level
	opts     []Option         // options used to create the logger (for cloning)
	caller   bool             // whether to display the caller
	callSkip int              // number of callers to skip until the actual caller
}

// New creates a new Logger
func New(opts ...Option) *Logger {
	cfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "lvl",
		NameKey:        "logger",
		TimeKey:        "ts",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	al := zap.NewAtomicLevelAt(zap.InfoLevel)
	zl := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.Lock(os.Stdout),
		al))

	l := &Logger{
		lvl:      InfoLevel,
		l:        zl,
		alvl:     &al,
		opts:     opts,
		caller:   true,
		callSkip: 3,
	}

	for _, o := range opts {
		o.apply(l)
	}

	return l
}

// Log logs a message at the lvl level. If there are formatArgs, it will format
// the message before writing it. It will include keyvals if present. Most
// package users will prefer to use the specialized helper functions (Info,
// Infof, ...)
func (l *Logger) Log(lvl Level, format string, formatArgs []interface{}, keyvals []interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lvl > l.lvl {
		return // we don't need to Log this
	}
	msg := format
	if msg == "" && len(formatArgs) > 0 {
		msg = fmt.Sprint(formatArgs...)
	} else if msg != "" && len(formatArgs) > 0 {
		msg = fmt.Sprintf(format, formatArgs...)
	}

	if ce := l.l.Check(tozaplevel(lvl), msg); ce != nil {
		fs := l.zapfields(keyvals)
		if lvl == FatalLevel {
			fs = append(fs, zap.Any("stacktrace", string(debug.Stack())))
		}
		if l.caller {
			fs = append(fs, zap.Any("caller", caller(l.callSkip)))
		}
		ce.Write(fs...)
	}
}

// Error logs an error message and the KV pairs
func (l *Logger) Error(msgOrError interface{}, keyvals ...interface{}) {
	l.Log(ErrorLevel, strOrErr(msgOrError), nil, keyvals)
}

// Errorf formats and logs an informational message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Log(ErrorLevel, format, args, nil)
}

// Warn logs a warning message and the KV pairs
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.Log(WarnLevel, msg, nil, keyvals)
}

// Warnf formats and logs a warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Log(WarnLevel, format, args, nil)
}

// Info logs an informational message and the KV pairs
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.Log(InfoLevel, msg, nil, keyvals)
}

// Infof formats and logs an informational message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Log(InfoLevel, format, args, nil)
}

// Debug logs a debug message and the KV pairs
func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.Log(DebugLevel, msg, nil, keyvals)
}

// Debugf formats and logs a debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Log(DebugLevel, format, args, nil)
}

// Fatal logs a fatal error. This logs in ErrorLevel as a simple error but it panics afterwards
func (l *Logger) Fatal(msgOrError interface{}, keyvals ...interface{}) {
	l.Log(FatalLevel, strOrErr(msgOrError), nil, keyvals)
}

// Fatalf will format and Log a fatal error message. It panics afterwards
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Log(FatalLevel, format, args, nil)
}

// SetLevel changes the Log level
func (l *Logger) SetLevel(lvl Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lvl = lvl
	l.alvl.SetLevel(tozaplevel(lvl))
}

// GetLevel gets the current Log level
func (l *Logger) GetLevel() Level { return l.lvl }

// WithCallSkip will return a logger with a new call skip number used to display
// the "caller" information in the logs
func (l *Logger) WithCallSkip(skip int) *Logger {
	nl := l.clone()
	nl.callSkip = skip
	return nl
}

func (l *Logger) clone() *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	nl := New(l.opts...)
	nl.alvl.SetLevel(l.alvl.Level())
	nl.lvl = l.lvl
	nl.with = append([]zap.Field(nil), l.with...)
	nl.l = nl.l.With(nl.with...)
	return nl
}

// With returns a new logger with the keyvals
func (l *Logger) With(keyvals ...interface{}) *Logger {
	nl := l.clone()
	zf := l.zapfields(keyvals)
	nl.with = append(nl.with, zf...) // we need to keep these for cloning
	nl.l = nl.l.With(zf...)
	return nl
}

// convert KV pairs to a slice of zap.Field
func (l *Logger) zapfields(keyvals []interface{}) []zap.Field {
	f := make([]zap.Field, 0, len(keyvals))
	for i := 0; i < len(keyvals); i += 2 {
		if i == len(keyvals)-1 {
			// dangling field...
			f = append(f, zap.String("PROBE_ERROR", fmt.Sprintf("found the key '%v' dangling. Did you mean to use (Debugf/Infof/etc) instead?", keyvals[i])))
			continue
		}
		key, ok := keyvals[i].(string)
		if !ok {
			// non-string key
			f = append(f, zap.String("PROBE_ERROR", fmt.Sprintf("invalid non-string key %v. Did you mean to use (Debugf/Infof/etc) instead?", keyvals[i])))
		}
		val := keyvals[i+1]
		f = append(f, zap.Any(key, val))
	}

	return f
}

// ReplaceGlobal replaces the global logger with a new one. This function is
// safe for concurrency
func ReplaceGlobal(l *Logger) {
	globalMu.Lock()
	globalL = l
	globalMu.Unlock()
}

func glog(lvl Level, format string, fmtArgs []interface{}, keyvals []interface{}) {
	globalMu.RLock()
	defer globalMu.RUnlock()
	globalL.Log(lvl, format, fmtArgs, keyvals)
}

// Error logs an error message and the KV pairs
func Error(msgOrError interface{}, keyvals ...interface{}) {
	glog(ErrorLevel, strOrErr(msgOrError), nil, keyvals)
}

// Errorf formats and logs an informational message
func Errorf(format string, args ...interface{}) {
	glog(ErrorLevel, format, args, nil)
}

// Warn logs a warning message and the KV pairs
func Warn(msg string, keyvals ...interface{}) {
	glog(WarnLevel, msg, nil, keyvals)
}

// Warnf formats and logs a warning message
func Warnf(format string, args ...interface{}) {
	glog(WarnLevel, format, args, nil)
}

// Info logs an informational message and the KV pairs
func Info(msg string, keyvals ...interface{}) {
	glog(InfoLevel, msg, nil, keyvals)
}

// Infof formats and logs an informational message
func Infof(format string, args ...interface{}) {
	glog(InfoLevel, format, args, nil)
}

// Debug logs a debug message and the KV pairs
func Debug(msg string, keyvals ...interface{}) {
	glog(DebugLevel, msg, nil, keyvals)
}

// Debugf formats and logs a debug message
func Debugf(format string, args ...interface{}) {
	glog(DebugLevel, format, args, nil)
}

// Fatal logs a fatal error message and panics
func Fatal(msgOrError interface{}, keyvals ...interface{}) {
	glog(FatalLevel, strOrErr(msgOrError), nil, keyvals)
}

// Fatalf formats and logs a fatal error message
func Fatalf(format string, args ...interface{}) {
	glog(FatalLevel, format, args, nil)
}

// SetLevel sets the logging level of the global logger
func SetLevel(lvl Level) {
	globalMu.Lock()
	globalL.SetLevel(lvl)
	globalMu.Unlock()
}

type call runtime.Frame

// we need to skip a number of callers to find the correct one, e.g. when a a
// log is called from metrics.go, we need to skip:
// 0. the very call below
// 1. the line that adds the caller to the list of fields
// 2. the line that writes the fields to zap
// 3. the Logger.Log call
func caller(skip int) call {
	// As of Go 1.9 we need room for up to three PC entries: the stackframe,
	// metadata, and target frame PC
	var pcs [3]uintptr

	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	_, _ = frames.Next() // skip
	frame, _ := frames.Next()

	return call(frame)
}

func (c call) String() string {
	if c == call(runtime.Frame{}) {
		return "(NOFUNC)"
	}

	file := filepath.Base(c.File)
	return fmt.Sprintf("%s:%d", file, c.Line)
}

func strOrErr(v interface{}) string {
	switch soe := v.(type) {
	case string:
		return soe
	case error:
		return soe.Error()
	default:
		panic(fmt.Sprintf("%T found instead of string or error", v))
	}
}
