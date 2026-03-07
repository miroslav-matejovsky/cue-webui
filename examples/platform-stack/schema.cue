// UI_Label: Deployment Platform
#Configuration: {
	// UI_Help: High-level release metadata.
	// UI_Columns: 3
	release: {
		name: string & =~"^[a-z0-9-]+$"
		version: string & =~"^v[0-9]+\\.[0-9]+\\.[0-9]+$" | *"v1.0.0"
		channel: "stable" | "candidate" | "nightly" | *"stable"
	}

	// UI_Help: How the application is exposed.
	// UI_Columns: 2
	exposure: {
		// UI_Widget: radio
		strategy: "ingress" | "loadbalancer" | "nodeport" | *"ingress"
		publicURL: string & =~"^https://.*$"
		enforceTLS: bool | *true
	}

	// UI_Help: Autoscaling and resources.
	// UI_Columns: 3
	compute: {
		replicas: int & >=1 & <=50 | *3
		cpuRequestMillicores: int & >=100 & <=8000 | *500
		memoryRequestMiB: int & >=128 & <=32768 | *1024
	}

	// UI_Help: Persistence and retention.
	// UI_Columns: 3
	storage: {
		class: "standard" | "premium-ssd" | "local-nvme"
		sizeGiB: int & >=10 & <=2048 | *100
		retentionDays: int & >=1 & <=365 | *30
	}

	// UI_Help: Operational context passed to the deployment team.
	// UI_Columns: 2
	metadata: {
		ownerEmail: string & =~"^[^@]+@[^@]+\\.[^@]+$"
		// UI_Widget: textarea
		changeSummary: string
		// UI_Readonly: true
		changeWindow: string | *"Sun 02:00-04:00 UTC"
		// UI_Hidden: true
		internalTicket: string | *"OPS-0000"
	}
}