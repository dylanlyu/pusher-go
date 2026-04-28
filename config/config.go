package config

import "net/http"

// BaseConfig holds connection settings shared by channels and beams clients.
type BaseConfig struct {
	HTTPClient *http.Client
	Host       string
}

// Option is a functional option for configuring a client of type T.
type Option[T any] func(*T)
