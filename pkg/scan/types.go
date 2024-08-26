package scan

// ScanOptions primarily is used for flags for the `scan` subcommand
type ScanOptions struct {
	FilterEventType FilterEventType
	FilterEvents    FilterEvents
	AlertFilters    AlertFilters

	GRPC         string
	Output       string
	RepoBranch   string
	PolicyAction string // Block or Audit
	PolicyEvent  string // ADDED or DELETED

	ShowProcessTree bool
	PolicyDryRun    bool
	StrictMode      bool
}

// Filter provides the basic filters for collection of data
type FilterEventType struct {
	System bool
	All    bool
}

// FilterEvents will filter events coming from KubeArmor
type FilterEvents struct {
	Network bool
	Process bool
	File    bool
}

// AlertFilters has options for filtering from alerts from KubeArmor
type AlertFilters struct {
	// Either file, network or process, so if someone has set
	// file, so we will ignore file
	IgnoreEvent string

	// Severity will filter only specific severity levels, from 1 to 10
	// so if its "--severity-level 5" then we show all the events from 5 and up
	SeverityLevel string
}
