package domain

// ServerConfig is the HTTP listener and access policy (WM-OPS-5, WM-SEC-1).
type ServerConfig struct {
	Host string
	Port int
}
