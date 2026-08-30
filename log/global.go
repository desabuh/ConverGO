package log

import "github.com/desabuh/convergo/config"

// a singleton log factory whose behavior is customized based on target textual identifier
type GlobalLoggerFactory interface {
	Create(id string) GlobalLogger
	CreateFromArgs(id string, args map[string]any) GlobalLogger
	config.ConfigExtractor[config.LoggerConfig]
}

// a global logger, provide a single api to provide a system-wide service over a single underlying logger
type GlobalLogger interface {
	Log(content string, args ...any)
}
