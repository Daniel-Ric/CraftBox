package minecraft

import "time"

const (
	DefaultResourcePackChunkSize = 1024 * 128

	DefaultResourcePackChunkSendDelay = 200 * time.Millisecond
)

type ResourcePackDeliveryConfig struct {
	ChunkSize uint32

	ChunkSendDelay time.Duration
}

func defaultResourcePackDeliveryConfig() ResourcePackDeliveryConfig {
	return ResourcePackDeliveryConfig{
		ChunkSize:      DefaultResourcePackChunkSize,
		ChunkSendDelay: DefaultResourcePackChunkSendDelay,
	}
}

func (config ResourcePackDeliveryConfig) normalized() ResourcePackDeliveryConfig {
	defaults := defaultResourcePackDeliveryConfig()
	if config.ChunkSize == 0 {
		config.ChunkSize = defaults.ChunkSize
	}
	if config.ChunkSendDelay == 0 {
		config.ChunkSendDelay = defaults.ChunkSendDelay
	} else if config.ChunkSendDelay < 0 {
		config.ChunkSendDelay = 0
	}
	return config
}
