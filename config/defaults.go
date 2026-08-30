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
	SEND_TIMEOUT       = 4 * time.Second
	BROADCAST_TIMEOUT  = 6 * time.Second

	LOG_DEFAULT_STREAM = os.Stdout
)

// default system transport config
var DefaultTransportConfig TransportConfig = TransportDefaultConfigExtractor{}.ExtractFrom(map[string]any{})

// default system broker config
var DefaultBrokerConfig BrokerConfig = BrokerDefaultConfigExtractor{}.ExtractFrom(map[string]any{})

// default system logger config
var DefaultLoggerConfig = LoggerDefaultConfigExtractor{}.ExtractFrom(map[string]any{})

// this interface define the behavior to return a default configuration
type ConfigExtractor[T SupportedConfiguration] interface {
	ExtractFrom(args map[string]any) T
}

type NeworkDefaultConfigExtractor struct{}

func (nmdc NeworkDefaultConfigExtractor) ExtractFrom(args map[string]any) NetworkModuleConfig {

	return NetworkModuleConfig{
		TransportConfig: TransportDefaultConfigExtractor{}.ExtractFrom(args),
		BrokerConfig:    BrokerDefaultConfigExtractor{}.ExtractFrom(args),
	}
}

type TransportDefaultConfigExtractor struct{}

func (tdc TransportDefaultConfigExtractor) ExtractFrom(args map[string]any) TransportConfig {
	return TransportConfig{
		InPairTimeout:   utils.GetOrDefault(args, "inPairTimeout", DEFAULT_IN_PAIR_TIMEOUT),
		OutPairtTimeout: utils.GetOrDefault(args, "outPairTimeout", DEFAULT_OUT_PAIR_TIMEOUT),
	}
}

type BrokerDefaultConfigExtractor struct{}

func (bdc BrokerDefaultConfigExtractor) ExtractFrom(args map[string]any) BrokerConfig {
	return BrokerConfig{
		QueueSize:        utils.GetOrDefault(args, "sessionQueueSize", SESSION_QUEUE_SIZE),
		SendTimeout:      utils.GetOrDefault(args, "sendTimeout", SEND_TIMEOUT),
		BroadcastTimeout: utils.GetOrDefault(args, "broadcastTimeout", BROADCAST_TIMEOUT),
	}
}

type LoggerDefaultConfigExtractor struct{}

func (ldc LoggerDefaultConfigExtractor) ExtractFrom(args map[string]any) LoggerConfig {

	return LoggerConfig{
		utils.GetOrDefault(args, "outStream", LOG_DEFAULT_STREAM),
	}
}
