package scan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	kaproto "github.com/kubearmor/KubeArmor/protobuf"
)

// SeverityLevel
type SeverityLevel struct {
	// Represents the severity level in integer
	Value int

	// Label represents the severity level in "words"
	Label string
}

var (
	SeverityInfo     = SeverityLevel{Value: 1, Label: "Info"}
	SeverityLow      = SeverityLevel{Value: 3, Label: "Low"}
	SeverityMedium   = SeverityLevel{Value: 5, Label: "Medium"}
	SeverityHigh     = SeverityLevel{Value: 7, Label: "High"}
	SeverityCritical = SeverityLevel{Value: 9, Label: "Critical"}
)

var severityLevels = []SeverityLevel{
	SeverityInfo,
	SeverityLow,
	SeverityMedium,
	SeverityHigh,
	SeverityCritical,
}

func GetSeverityLevel(value int) SeverityLevel {
	for _, level := range severityLevels {
		if value <= level.Value {
			return level
		}
	}

	return SeverityCritical
}

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
	Severity SeverityLevel `json:"severity"`
}

// AlertProcessor represents alerts cache and filters
type AlertProcessor struct {
	// alerts caches the alerts
	// each alert is stored against a PID
	// with a 'key' to make sure that we only store
	// unique alerts for a given PID
	alerts map[int32]map[string]Alert

	// filters are used to filter out specific alerts
	filters AlertFilters
}

// NewAlertProcessor returns new instance of alerts processor
func NewAlertProcessor(filters AlertFilters) *AlertProcessor {
	return &AlertProcessor{
		alerts:  make(map[int32]map[string]Alert),
		filters: filters,
	}
}

// ProcessAlerts processes alerts 
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

		severityValue, _ := strconv.Atoi(kaAlert.Severity)
		alert := Alert{
			PolicyName:  kaAlert.PolicyName,
			Operation:   kaAlert.Operation,
			PID:         kaAlert.PID,
			ProcessName: kaAlert.ProcessName,
			Command:     getActualProcessName(kaAlert.Source),
			Message:     kaAlert.Message,
			Tags:        ap.processTags(kaAlert),
			Severity:    GetSeverityLevel(severityValue),
		}

		// Create a unique key for the alert
		alertKey := fmt.Sprintf("%s-%s-%s-%s", alert.PolicyName, alert.Operation, alert.ProcessName, alert.Message)

		if _, exists := ap.alerts[kaAlert.PID]; !exists {
			ap.alerts[kaAlert.PID] = make(map[string]Alert)
		}
		ap.alerts[kaAlert.PID][alertKey] = alert
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

	alertsBySeverity := make(map[SeverityLevel][]Alert)
	for _, alertMap := range ap.alerts {
		for _, alert := range alertMap {
			alertsBySeverity[alert.Severity] = append(alertsBySeverity[alert.Severity], alert)
		}
	}

	// Sort severity levels
	sortedSeverities := make([]SeverityLevel, 0, len(alertsBySeverity))
	for severity := range alertsBySeverity {
		sortedSeverities = append(sortedSeverities, severity)
	}
	sort.Slice(sortedSeverities, func(i, j int) bool {
		return sortedSeverities[i].Value > sortedSeverities[j].Value
	})

	for _, severity := range sortedSeverities {
		alerts := alertsBySeverity[severity]
		sb.WriteString(fmt.Sprintf("### %s (%d alerts)\n\n", severity.Label, len(alerts)))
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
