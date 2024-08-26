package scan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	kaproto "github.com/kubearmor/KubeArmor/protobuf"
)

type Alert struct {
	// Name of the policy
	PolicyName string `json:"policyName"`

	// Operation
	Operation string `json:"operation"`

	// PID
	PID int32 `json:"pid"`

	// Process name
	ProcessName string `json:"processName"`

	// Command
	Command string `json:"command"`

	// Message
	Message string `json:"message"`

	// Tags
	Tags []string `json:"tags"`

	// Severity level
	Severity string `json:"severity"`
}

type AlertProcessor struct {
	alerts  map[int32]Alert
	filters AlertFilters
}

func NewAlertProcessor(filters AlertFilters) *AlertProcessor {
	return &AlertProcessor{
		alerts:  make(map[int32]Alert),
		filters: filters,
	}
}

func (ap *AlertProcessor) ProcessAlerts(segregatedData *SegregatedData) {
	ap.processAlertGroup(segregatedData.Alerts.Network, "network")
	ap.processAlertGroup(segregatedData.Alerts.File, "file")
	ap.processAlertGroup(segregatedData.Alerts.Process, "process")
}

func (ap *AlertProcessor) processAlertGroup(alerts []kaproto.Alert, eventType string) {
	for _, kaAlert := range alerts {
		if !ap.shouldProcessAlerts(kaAlert, eventType) {
			continue
		}

		alert := Alert{
			PolicyName:  kaAlert.PolicyName,
			Operation:   kaAlert.Operation,
			PID:         kaAlert.PID,
			ProcessName: kaAlert.ProcessName,
			Command:     getActualProcessName(kaAlert.Source),
			Message:     kaAlert.Message,
			Tags:        ap.processTags(kaAlert),
			Severity:    kaAlert.Severity,
		}
		ap.alerts[kaAlert.PID] = alert
	}
}

func (ap *AlertProcessor) shouldProcessAlerts(kaAlert kaproto.Alert, eventType string) bool {
	if ap.filters.IgnoreEvent == eventType {
		return false
	}

	if ap.filters.SeverityLevel != "" {
		filterLevel, err := strconv.Atoi(ap.filters.SeverityLevel)
		if err != nil {
			fmt.Printf("invalid alert severity: %s\n", err.Error())
			return false
		}

		alertLevel, err := strconv.Atoi(kaAlert.Severity)
		if err != nil {
			fmt.Printf("invalid alert severity: %s\n", err.Error())
			return false
		}

		if alertLevel < filterLevel {
			return false
		}
	}

	return true
}

func (ap *AlertProcessor) processTags(kaAlert kaproto.Alert) []string {
	if len(kaAlert.ATags) > 0 {
		return kaAlert.ATags
	}
	return strings.Split(kaAlert.Tags, ",")
}

// GenerateJSON generates a JSON representation of the alerts
func (ap *AlertProcessor) GenerateJSON() ([]byte, error) {
	return json.MarshalIndent(ap.alerts, "", "  ")
}

// GenerateMarkdownTable generates a fancy markdown table of alerts
func (ap *AlertProcessor) GenerateMarkdownTable() string {
	var sb strings.Builder

	alertsBySeverity := make(map[string][]Alert)
	for _, alert := range ap.alerts {
		alertsBySeverity[alert.Severity] = append(alertsBySeverity[alert.Severity], alert)
	}

	severities := make([]string, 0, len(alertsBySeverity))
	for severity := range alertsBySeverity {
		severities = append(severities, severity)
	}
	sort.Slice(severities, func(i, j int) bool {
		a, _ := strconv.Atoi(severities[i])
		b, _ := strconv.Atoi(severities[j])
		return a > b
	})

	for _, severity := range severities {
		alerts := alertsBySeverity[severity]
		sb.WriteString(fmt.Sprintf("## Severity %s (%d alerts)\n\n", severity, len(alerts)))
		sb.WriteString("<details>\n<summary>Click to expand</summary>\n\n")

		sb.WriteString("| 📜 Policy Name | 🔧 Operation | 🔢 PID | ⚡ Command | 💻 Process Name | 📣 Message | 🏷️ Tags |\n")
		sb.WriteString("|----------------|--------------|--------|------------|----------------|-----------|--------|\n")

		for _, alert := range alerts {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s | %s | %s |\n",
				alert.PolicyName,
				alert.Operation,
				alert.PID,
				alert.Command,
				alert.ProcessName,
				alert.Message,
				strings.Join(alert.Tags, ", ")))
		}

		sb.WriteString("\n</details>\n\n")
	}

	return sb.String()
}
