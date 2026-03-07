// UI_Label: Platform Control Plane
// UI_Navigation: tabs
#Configuration: {
	// UI_Label: Edge Network
	// UI_Help: Listener, TLS, and ingress settings.
	network: {
		// UI_Navigation: tabs
		listeners: {
			// UI_Label: Public HTTPS
			public: {
				host: string | *"api.example.com"
				port: int & >=1 & <=65535 | *443
				protocol: "https" | "h2"
			}

			// UI_Label: Internal gRPC
			internal: {
				host: string | *"control-plane.svc.cluster.local"
				port: int & >=1 & <=65535 | *8443
				protocol: "grpc" | "grpcs"
			}
		}

		// UI_Navigation: tabs
		tls: {
			// UI_Label: Certificates
			certificates: {
				certFile: string
				keyFile: string
				caFile:   string
			}

			// UI_Label: Policy
			policy: {
				clientAuth: "none" | "request" | "require"
				minVersion: "1.2" | "1.3" | *"1.3"
			}
		}
	}

	// UI_Label: Runtime
	// UI_Help: Feature switches and workload policy.
	runtime: {
		// UI_Navigation: tabs
		features: {
			canary: {
				enabled: bool | *true
				strategy: "weighted" | "header" | "shadow"
			}
			jobs: {
				workerCount: int & >=1 & <=64 | *8
				retryPolicy: "never" | "linear" | "exponential"
			}
		}

		// UI_Help: Notes for operators during rollout.
		// UI_Widget: textarea
		notes: string | *"Roll out to one region before expanding globally."
	}
}