package config

import (
	"os"
	"time"
)

type TransportConfig struct {
	InPairTimeout   time.Duration
	OutPairtTimeout time.Duration
}

type BrokerConfig struct {
	QueueSize        int
	SendTimeout      time.Duration
	BroadcastTimeout time.Duration
}

type LoggerConfig struct {
	OutStream *os.File
}

type NetworkModuleConfig struct {
	TransportConfig
	BrokerConfig
}
