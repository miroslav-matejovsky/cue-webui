// Connection settings for the server.
#Connection: {
	// UI_Label: Server Address
	// UI_Help: Hostname or IP address to listen on
	// UI_Placeholder: e.g. 0.0.0.0
	address: string

	// UI_Help: TCP port number (1-65535)
	// UI_Placeholder: 8080
	// UI_Min: 1
	// UI_Max: 65535
	port: int

	// UI_Widget: select
	// UI_Options: http, https, tcp, udp
	// UI_Help: Network protocol to use
	protocol: string
}

// Logging configuration.
#Logging: {
	// UI_Widget: select
	// UI_Options: debug, info, warn, error
	// UI_Help: Minimum log level to output
	level: string

	// UI_Widget: select
	// UI_Options: json, text
	// UI_Help: Log output format
	format: string

	// UI_Placeholder: stdout
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
