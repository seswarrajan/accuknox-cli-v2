package scan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/accuknox/accuknox-cli-v2/pkg/scan/policy"
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

	// Operation type which can be either network, file or process
	Operation string `json:"operation"`

	// PID of the process for which the alert is generated
	PID int32 `json:"pid"`

	// Process name for which the alert is generated
	ProcessName string `json:"processName"`

	// Command executed by the process
	Command string `json:"command"`

	// Message as given in the policy
	Message string `json:"message"`

	// Tags as given in the policy
	Tags []string `json:"tags"`

	// Severity level as given in the policy
	Severity SeverityLevel `json:"severity"`

	// Action can either be blocked or audit
	Action string `json:"action"`
}

// AlertProcessor represents alerts cache and filters
type AlertProcessor struct {
	// alerts caches the alerts
	// each alert is stored against a PID
	// with a 'key' to make sure that we only store
	// unique alerts for a given PID
	alerts map[int32]map[string]AlertPair

	// filters are used to filter out specific alerts
	filters AlertFilters

	// Policy reader
	policyReader *policy.PolicyReader
}

// AlertPair contains custom and raw alert
type AlertPair struct {
	// CustomAlert
	CustomAlert Alert

	// KAAlert
	KAAlert kaproto.Alert
}

// NewAlertProcessor returns new instance of alerts processor
func NewAlertProcessor(filters AlertFilters) *AlertProcessor {
	policyReader, err := policy.NewPolicyReader()
	if err != nil {
		fmt.Printf("Failed to init policy reader, policies will not be shown in final report: %v\n", err)
	}

	return &AlertProcessor{
		alerts:       make(map[int32]map[string]AlertPair),
		filters:      filters,
		policyReader: policyReader,
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
		customAlert := Alert{
			PolicyName:  kaAlert.PolicyName,
			Operation:   kaAlert.Operation,
			PID:         kaAlert.PID,
			ProcessName: kaAlert.ProcessName,
			Command:     getActualProcessName(kaAlert.Source),
			Message:     kaAlert.Message,
			Tags:        ap.processTags(kaAlert),
			Severity:    GetSeverityLevel(severityValue),
			Action:      kaAlert.Action,
		}

		// Create a unique key for the alert
		alertKey := fmt.Sprintf("%s-%s-%s-%s-%s", customAlert.PolicyName, customAlert.Operation, customAlert.ProcessName, customAlert.Message, customAlert.Action)

		if _, exists := ap.alerts[kaAlert.PID]; !exists {
			ap.alerts[kaAlert.PID] = make(map[string]AlertPair)
		}
		ap.alerts[kaAlert.PID][alertKey] = AlertPair{
			CustomAlert: customAlert,
			KAAlert:     kaAlert,
		}
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

	alertsBySeverity := make(map[SeverityLevel][]AlertPair)
	for _, alertMap := range ap.alerts {
		for _, alertPair := range alertMap {
			alertsBySeverity[alertPair.CustomAlert.Severity] = append(alertsBySeverity[alertPair.CustomAlert.Severity], alertPair)
		}
	}

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

		for _, alertPair := range alerts {
			alert := alertPair.CustomAlert

			// We need to write table headers for each row, since we also show up the collapsible view of KubeArmor JSON alert
			sb.WriteString("| 📜 Policy Name | 🔧 Operation | 🔢 PID | ⚡ Command | 💻 Process Name | 📣 Message | 🏷️ Tags | 🛡️ Action |\n")
			sb.WriteString("|----------------|--------------|--------|------------|-----------------|------------|---------|------------|\n")

			// Write table row
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s | %s | %s | %s |\n",
				alert.PolicyName,
				alert.Operation,
				alert.PID,
				alert.Command,
				alert.ProcessName,
				alert.Message,
				strings.Join(alert.Tags, ", "),
				alert.Action))

			// Adding collapsible JSON immediately after the row
			jsonAlert, err := json.MarshalIndent(alertPair.KAAlert, "", "  ")
			if err != nil {
				sb.WriteString(fmt.Sprintf("Error marshaling alert to JSON: %v\n", err))
			} else {
				sb.WriteString("\n<details>\n<summary>Click to view complete alert</summary>\n\n```json\n")
				sb.WriteString(string(jsonAlert))
				sb.WriteString("\n```\n</details>\n\n")
			}

			optimizedYAML, err := ap.policyReader.GetOptimizedPolicyYAML(alert.PolicyName)
			if err != nil {
				sb.WriteString("Failed to get the associated policy\n")
			} else {
				sb.WriteString("#### Related Policy Details\n\n")
				sb.WriteString("<details>\n<summary>Click to view policy YAML</summary>\n\n```yaml\n")
				sb.WriteString(optimizedYAML)
				sb.WriteString("\n```\n</details>\n\n")
			}

			// Seperator
			sb.WriteString("\n---\n\n")
		}

		sb.WriteString("</details>\n\n")
	}

	return sb.String()
}
