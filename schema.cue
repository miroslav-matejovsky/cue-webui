// #Configuration defines the top-level server configuration.
#Configuration: {
	// address is the hostname or IP address the server listens on.
	// Example: "0.0.0.0" to listen on all interfaces, or "127.0.0.1" for localhost only.
	address: string

	// port is the TCP port number the server binds to.
	// Must be a valid port in the range 1–65535.
	port: int

	// protocol specifies the network protocol used by the server.
	// Typical values: "http", "https", "tcp", "udp".
	protocol: string
}

// Additional constraints and notes for #Configuration.
#Configuration: {
	// address must be a non-empty string; no validation of DNS resolution is performed here.
	address: string

	// port values below 1024 require elevated privileges on most operating systems.
	port: int

	// protocol is case-sensitive and must match the transport layer implementation.
	protocol: string
}
