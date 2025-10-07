package app

import "time"

type Config struct {
	EByteAddress   string
	ListenAddress  string
	ReconnectDelay time.Duration
	LogLevel       string
}
