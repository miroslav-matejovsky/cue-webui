// #Configuration defines the top-level server configuration.
// UI_Rows: 3
// UI_Columns: 1
#Connection: {
	// UI_Label: Address
	// UI_Help: The network address the server listens on.
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

#Logging: {
	// level is the minimum log level to output.
	// Common levels: "debug", "info", "warn", "error".
	level: string

	// format specifies the log output format.
	// Examples: "json", "text".
	format: string

	// output defines where logs are written.
	// Examples: "stdout", "stderr", or a file path like "/var/log/server.log".
	output: string
}

#Configuration: {
	// connection holds the server's network configuration.
	connection: #Connection

	// logging defines the server's logging settings.
	logging: #Logging
}
