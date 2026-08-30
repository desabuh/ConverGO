package log

import (
	"io"

	"github.com/desabuh/convergo/config"
)

// A formattable mutex logger, provide along with each mutual exclusion system log an identifier prefix
type FormattedMutexLogger struct {
	logger   *MutexLogger
	out      io.Writer
	idPrefix string
}

func (l *FormattedMutexLogger) Log(content string, args ...any) {
	l.logger.PrefNlog(l.idPrefix, l.out, content, args...)
}

type FormattedMutexLoggerFactory struct {
	logger *MutexLogger
	config.ConfigExtractor[config.LoggerConfig]
}

func NewMutexLoggerFactory(configurable config.ConfigExtractor[config.LoggerConfig]) *FormattedMutexLoggerFactory {
	return &FormattedMutexLoggerFactory{
		logger:          NewMutexLogger(),
		ConfigExtractor: configurable,
	}
}

func (f *FormattedMutexLoggerFactory) Create(id string) GlobalLogger {

	loggerConfig := f.ExtractFrom(map[string]any{})

	return &FormattedMutexLogger{
		logger:   f.logger,
		out:      loggerConfig.OutStream,
		idPrefix: id,
	}
}

func (f *FormattedMutexLoggerFactory) CreateFromArgs(id string, args map[string]any) GlobalLogger {
	loggerConfig := f.ExtractFrom(args)

	return &FormattedMutexLogger{
		logger:   f.logger,
		out:      loggerConfig.OutStream,
		idPrefix: id,
	}
}
