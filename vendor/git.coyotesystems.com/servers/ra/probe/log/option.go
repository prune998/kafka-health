package log

// Option configures the logger
type Option interface {
	apply(*Logger)
}

// helper LoggerOption implementation to quickly define new options
type optionFunc func(*Logger)

func (f optionFunc) apply(l *Logger) {
	f(l)
}

// WithLogLevel sets the logging level
func WithLogLevel(lvl Level) Option {
	return optionFunc(func(l *Logger) {
		l.lvl = lvl
		l.alvl.SetLevel(tozaplevel(lvl))
	})
}

// WithError sets the logging level to Debug
func WithError() Option { return WithLogLevel(ErrorLevel) }

// WithWarn sets the logging level to Warn
func WithWarn() Option { return WithLogLevel(WarnLevel) }

// WithInfo sets the logging level to Info
func WithInfo() Option { return WithLogLevel(InfoLevel) }

// WithDebug sets the logging level to Debug
func WithDebug() Option { return WithLogLevel(DebugLevel) }

// WithDisplayCaller sets whether to display the calling filename and line
func WithDisplayCaller(display bool) Option {
	return optionFunc(func(l *Logger) {
		l.caller = display
		// number of calls to skip (the very call to "call", the logging
		// function, etc) so we can find the correct caller that we want to log.
		// By default we skip 3 which skips the callers in logger.go. See the
		// docs in logger.go for more.
		l.callSkip = 3
	})
}
