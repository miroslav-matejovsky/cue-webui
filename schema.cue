// Connection settings for the server.
#Connection: {
	// UI_Label: Server Address
	// UI_Help: Hostname or IP address to listen on
	address: string

	// UI_Help: TCP port number (1-65535)
	port: int & >=1 & <=65535

	// UI_Help: Network protocol to use
	protocol: "http" | "https" | "tcp" | "udp"
}

// Logging configuration.
#Logging: {
	// UI_Help: Minimum log level to output
	level: "debug" | "info" | "warn" | "error"

	// UI_Help: Log output format
	format: "json" | "text"

	// UI_Help: Log output destination (stdout, stderr, or file path)
	output: string
}

// Top-level server configuration.
// UI_Label: Server Configuration
#Configuration: {
	// UI_Help: Network and protocol settings
	// UI_Columns: 3
	connection: #Connection

	// UI_Help: Logging preferences
	// UI_Columns: 3
	logging: #Logging
}
