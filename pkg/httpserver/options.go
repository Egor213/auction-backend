package httpserver

import "time"

type Option func(*HttpServer)

func Address(address string) Option {
	return func(hs *HttpServer) {
		hs.server.Addr = address
	}
}

func ReadTimeout(timeout time.Duration) Option {
	return func(s *HttpServer) {
		s.server.ReadTimeout = timeout
	}
}

func WriteTimeout(timeout time.Duration) Option {
	return func(s *HttpServer) {
		s.server.WriteTimeout = timeout
	}
}

func ShutdownTimeout(timeout time.Duration) Option {
	return func(s *HttpServer) {
		s.shutdownTimeout = timeout
	}
}
