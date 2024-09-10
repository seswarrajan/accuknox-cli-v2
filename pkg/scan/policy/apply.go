package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/accuknox/accuknox-cli-v2/pkg/common"
	jsoniter "github.com/json-iterator/go"
	katypes "github.com/kubearmor/KubeArmor/KubeArmor/types"
	kaproto "github.com/kubearmor/KubeArmor/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/yaml"
)

const (
	ActionBlock = "Block"
	ActionAudit = "Audit"
)

// SkipPolicy has a set of policies that are not applied, but they will
// applied if they are ran in `strict` mode. These policies are not applied
// since they tend to generate a lot of alerts.
var SkipPolicy = map[string]bool{
	"hsp-nist-ca-9-audit-untrusted-read-on-sensitive-files":              true,
	"hsp-cm-1-configuration-management-policy-and-procedures":            true,
	"hsp-block-stig-ubuntu-20-010427-lib":                                true,
	"hsp-cve-2019-13139-docker-build":                                    true,
	"hsp-ca-7-4-continuous-monitoring-automation-support-for-monitoring": true,
	"hsp-mitre-persistence-bash-profile-audit":                           true,
	"hsp-mitre-ptrace-syscall":                                           true,
	"hsp-mitre-t1053-003-scheduled-task-job-crontab":                     true,
	"hsp-nist-au-3-audit-etc-dir":                                        true,
	"hsp-cis-1-1-9-api-cni-files":                                        true,
}

// Apply policies via gRPC
type Apply struct {
	// grpc connection
	conn *grpc.ClientConn

	// grpc connection string
	connString string

	// policy cache (this is public because we need the policy cache)
	Policies *GetPolicy

	// policy service client
	policyService kaproto.PolicyServiceClient

	// Name of the host
	hostname string

	// action (Audit or Block)
	action string

	// event type (ADDED or DELETED)
	event string

	// dryrun
	dryrun bool

	// run in strict mode
	strictMode bool

	// generated policies
	generatedPolicies [][]byte

	// user defined policies path
	userPoliciesPath string

	// user defined policies
	userPolicies []*KubeArmorPolicy
}

// NewApplier will instantiate the policy applier
func NewApplier(connString, zipURL, hostname, action, event, userPoliciesPath string, strictMode, dryrun bool) *Apply {
	return &Apply{
		connString:       connString,
		Policies:         NewGenerator(zipURL),
		hostname:         hostname,
		action:           action,
		event:            event,
		dryrun:           dryrun,
		strictMode:       strictMode,
		userPoliciesPath: userPoliciesPath,
	}
}

// Apply connects with gRPC and starts to apply the policies
func (a *Apply) Apply() error {
	err := a.connectToGRPC()
	if err != nil {
		return err
	}

	a.policyService = kaproto.NewPolicyServiceClient(a.conn)

	err = a.loadUserPolicies()
	if err != nil {
		return fmt.Errorf("failed to load user-defined policies: %v", err)
	}

	if len(a.userPolicies) > 0 {
		fmt.Println("Found user defined policies")

		err = a.handleUserPolicy()
		if err != nil {
			return fmt.Errorf("failed to handle user policies: %v", err)
		}
	}

	err = a.Policies.FetchTemplates()
	if err != nil {
		return fmt.Errorf("failed to fetch policy templates: %v", err.Error())
	}

	return a.handlePolicies()
}

func (a *Apply) handlePolicies() error {
	if a.strictMode {
		fmt.Println("Running in strict mode, all the policies will be applied")
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(a.Policies.PolicyCache))

	for _, policy := range a.Policies.PolicyCache {
		wg.Add(1)
		go func(p *KubeArmorPolicy) {
			defer wg.Done()
			a.modifyPolicy(p, false)
			if err := a.processPolicy(p); err != nil {
				errorChan <- err
			}
		}(policy)
	}

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// omitEmpty removes empty fields from a map recursively
// this is a custom omission mechanism because k8s yaml engine
// just doesn't works and fails to exclude empty fields
func omitEmpty(m map[string]any) map[string]any {
	for k, v := range m {
		if v == nil {
			delete(m, k)
			continue
		}
		switch v := v.(type) {
		case map[string]any:
			m[k] = omitEmpty(v)
			if len(m[k].(map[string]any)) == 0 {
				delete(m, k)
			}
		case []any:
			if len(v) == 0 {
				delete(m, k)
			}
		case string:
			if v == "" {
				delete(m, k)
			}
		case int:
			if v == 0 {
				delete(m, k)
			}
		case float64:
			if v == 0 {
				delete(m, k)
			}
		case bool:
			if !v {
				delete(m, k)
			}
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Struct {
				m[k] = omitEmpty(structToMap(v))
				if len(m[k].(map[string]any)) == 0 {
					delete(m, k)
				}
			}
		}
	}
	return m
}

// structToMap converts a struct to a map
func structToMap(obj any) map[string]any {
	out := make(map[string]any)
	j, _ := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(obj)
	_ = jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(j, &out)
	return out
}

// processPolicy basically does the following
// After skipping certain policies, we
// first, we modify the whole policy to change few fields
// second, it applies a transformation by converting the policy to map via json
// third, we omit the empty fields
// fourth, we convert the policy back to json bytes and then from json to yaml
// finally, it sends the policy to be applied via gRPC
func (a *Apply) processPolicy(policy *KubeArmorPolicy) error {
	if a.event == "ADDED" && !a.dryrun && !a.strictMode {
		if _, ok := SkipPolicy[policy.Metadata.Name]; ok {
			fmt.Printf("Omiting policy addition\n")
			return nil
		}
	}

	// Convert policy to map and omit empty fields
	policyMap := structToMap(policy)
	cleanedPolicyMap := omitEmpty(policyMap)

	// Marshal the cleaned map back to JSON
	cleanedJSON, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(cleanedPolicyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal cleaned policy: %s", err.Error())
	}

	// Convert JSON to YAML
	policyBytes, err := yaml.JSONToYAML(cleanedJSON)
	if err != nil {
		return fmt.Errorf("failed to convert JSON to YAML: %s", err.Error())
	}

	if a.dryrun {
		a.generatedPolicies = append(a.generatedPolicies, policyBytes)
		return nil
	}

	var hostPolicy katypes.K8sKubeArmorHostPolicy
	err = jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(cleanedJSON, &hostPolicy)
	if err != nil {
		return fmt.Errorf("failed to unmarshal to json: %s", err.Error())
	}

	policyEvent := katypes.K8sKubeArmorHostPolicyEvent{
		Type:   a.event,
		Object: hostPolicy,
	}

	policyEventBytes, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(policyEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal the policy event: %s", err.Error())
	}

	return a.applyPolicy(policyEventBytes)
}

func (a *Apply) modifyPolicy(policy *KubeArmorPolicy, byUser bool) {
	if policy.Spec.NodeSelector.MatchLabels == nil {
		policy.Spec.NodeSelector.MatchLabels = make(map[string]string)
	}

	// Always set the hostname to a.hostname, regardless of what's in the policy
	policy.Spec.NodeSelector.MatchLabels["kubearmor.io/hostname"] = a.hostname

	// Remove kubernetes.io/hostname if it exists
	delete(policy.Spec.NodeSelector.MatchLabels, "kubernetes.io/hostname")

	// Return since we don't want to change anything else in a user defined policy
	if byUser {
		return
	}

	newAction := a.action

	if policy.Spec.Action != "" {
		policy.Spec.Action = newAction
	}

	// Update Process actions
	updateProcessActions(&policy.Spec.Process, newAction)

	// Update File actions
	updateFileActions(&policy.Spec.File, newAction)

	// Update Network actions
	updateNetworkActions(&policy.Spec.Network, newAction)

	// Update Capabilities actions
	updateCapabilitiesActions(&policy.Spec.Capabilities, newAction)
}

func updateProcessActions(process *ProcessType, newAction string) {
	if process.Action != "" {
		process.Action = newAction
	}
	for i := range process.MatchPaths {
		if process.MatchPaths[i].Action != "" {
			process.MatchPaths[i].Action = newAction
		}
	}
	for i := range process.MatchDirectories {
		if process.MatchDirectories[i].Action != "" {
			process.MatchDirectories[i].Action = newAction
		}
	}
	for i := range process.MatchPatterns {
		if process.MatchPatterns[i].Action != "" {
			process.MatchPatterns[i].Action = newAction
		}
	}
}

func updateFileActions(file *FileType, newAction string) {
	if file.Action != "" {
		file.Action = newAction
	}
	for i := range file.MatchPaths {
		if file.MatchPaths[i].Action != "" {
			file.MatchPaths[i].Action = newAction
		}
	}
	for i := range file.MatchDirectories {
		if file.MatchDirectories[i].Action != "" {
			file.MatchDirectories[i].Action = newAction
		}
	}
	for i := range file.MatchPatterns {
		if file.MatchPatterns[i].Action != "" {
			file.MatchPatterns[i].Action = newAction
		}
	}
}

func updateNetworkActions(network *NetworkType, newAction string) {
	if network.Action != "" {
		network.Action = newAction
	}
	for i := range network.MatchProtocols {
		if network.MatchProtocols[i].Action != "" {
			network.MatchProtocols[i].Action = newAction
		}
	}
}

func updateCapabilitiesActions(capabilities *CapabilitiesType, newAction string) {
	if capabilities.Action != "" {
		capabilities.Action = newAction
	}
	for i := range capabilities.MatchCapabilities {
		if capabilities.MatchCapabilities[i].Action != "" {
			capabilities.MatchCapabilities[i].Action = newAction
		}
	}
}

func (a *Apply) applyPolicy(policyBytes []byte) error {
	req := kaproto.Policy{
		Policy: policyBytes,
	}

	resp, err := a.policyService.HostPolicy(context.Background(), &req)
	if err != nil {
		return fmt.Errorf("failed to send policy over grpc: %s", err.Error())
	}

	fmt.Printf("Policy %s \n", resp.Status)
	return nil
}

func (a *Apply) connectToGRPC() error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32 * 10e6)),
	}

	if a.connString == "" {
		a.connString = common.KubeArmorGRPCAddress
	}
	connection, err := grpc.Dial(a.connString, opts...)
	if err != nil {
		return err
	}

	a.conn = connection
	return nil
}

func (a *Apply) loadUserPolicies() error {
	if a.userPoliciesPath == "" {
		return nil
	}

	content, err := common.CleanAndRead(a.userPoliciesPath)
	if err != nil {
		return fmt.Errorf("failed to read user policies file: %v", err)
	}

	policyYAMLs := strings.Split(string(content), "---")

	for _, policyYAML := range policyYAMLs {
		policyYAML = strings.TrimSpace(policyYAML)
		if policyYAML == "" {
			continue
		}

		var policy KubeArmorPolicy
		err := yaml.Unmarshal([]byte(policyYAML), &policy)
		if err != nil {
			return fmt.Errorf("failed to parse user policy: %v", err)
		}

		a.userPolicies = append(a.userPolicies, &policy)
	}

	return nil
}

func (a *Apply) handleUserPolicy() error {
	for _, policy := range a.userPolicies {
		// Just modify the policy to write the current hostname
		a.modifyPolicy(policy, true)

		err := a.processPolicy(policy)
		if err != nil {
			return fmt.Errorf("failed to process user-defined policy: %v", err)
		}
	}

	return nil
}

// SavePolicies saves the generated policies to a file in YAML format
func (a *Apply) SavePolicies(filePath string) error {
	if !a.dryrun || len(a.generatedPolicies) == 0 {
		return fmt.Errorf("no policies to save or not in dry run mode")
	}

	file, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err.Error())
	}
	defer file.Close()

	for i, policy := range a.generatedPolicies {
		if i > 0 {
			_, err = file.WriteString("---\n")
			if err != nil {
				return fmt.Errorf("failed to write separator: %s", err.Error())
			}
		}

		_, err = file.Write(policy)
		if err != nil {
			return fmt.Errorf("failed to write policy: %s", err.Error())
		}

		_, err = file.WriteString("\n")
		if err != nil {
			return fmt.Errorf("failed to write newline: %s", err.Error())
		}
	}

	return nil
}
