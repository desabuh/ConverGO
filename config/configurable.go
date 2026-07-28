package config

import (
	"os"
	"time"

	"github.com/desabuh/convergo/utils"
)

// default system config
var (
	DEFAULT_IN_PAIR_TIMEOUT  = 10 * time.Second
	DEFAULT_OUT_PAIR_TIMEOUT = 10 * time.Second

	SESSION_QUEUE_SIZE = 10
	SEND_TIMEOUT       = 3 * time.Second
	BROADCAST_TIMEOUT  = 5 * time.Second

	LOG_DEFAULT_STREAM = os.Stdout
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

type Configurable[T any] interface {
	ExtractFrom(args map[string]any) T
}

type NeworkModuleDefaultConfigurable struct{}

func (nmdc NeworkModuleDefaultConfigurable) ExtractFrom(args map[string]any) NetworkModuleConfig {

	return NetworkModuleConfig{
		TransportConfig{
			InPairTimeout:   utils.GetOrDefault(args, "inPairTimeout", DEFAULT_IN_PAIR_TIMEOUT),
			OutPairtTimeout: utils.GetOrDefault(args, "outPairTimeout", DEFAULT_OUT_PAIR_TIMEOUT),
		},
		BrokerConfig{
			QueueSize:        utils.GetOrDefault(args, "sessionQueueSize", SESSION_QUEUE_SIZE),
			SendTimeout:      utils.GetOrDefault(args, "sendTimeout", SEND_TIMEOUT),
			BroadcastTimeout: utils.GetOrDefault(args, "broadcastTimeout", BROADCAST_TIMEOUT),
		},
	}
}

type LoggerDefaultConfigurable struct{}

func (ldc LoggerDefaultConfigurable) ExtractFrom(args map[string]any) LoggerConfig {

	return LoggerConfig{
		utils.GetOrDefault(args, "outStream", LOG_DEFAULT_STREAM),
	}
}
