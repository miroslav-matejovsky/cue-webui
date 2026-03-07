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
