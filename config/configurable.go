package config

//currently supported configuration in the system, update this signature in case of system extension
type SupportedConfiguration interface {
	TransportConfig | LoggerConfig | BrokerConfig | NetworkModuleConfig
}

//a signature to embed in an entity with config setting capabilities, underlying T config should be in SupportedConfiguration signature
type Configurable[T SupportedConfiguration] interface {
	SetConfig(config T)
}

func WithConfig[T SupportedConfiguration](configurable Configurable[T], config T) {
	configurable.SetConfig(config)
}
