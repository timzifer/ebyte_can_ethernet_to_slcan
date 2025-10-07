package app

import "time"

// Config collects runtime settings for the bridge.
type Config struct {
	EByteAddress   string
	ListenAddress  string
	ReconnectDelay time.Duration
	LogLevel       string
	BusBitrate     uint32
}
