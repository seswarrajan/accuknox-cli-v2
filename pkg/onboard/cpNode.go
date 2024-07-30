package onboard

import (
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/accuknox/accuknox-cli-v2/pkg/common"
	"golang.org/x/mod/semver"
)

type agentConfigMeta struct {
	agentName                string
	configPath               string
	configTemplateString     string
	kmuxConfigPath           string
	kmuxConfigTemplateString string
}

func InitCPNodeConfig(cc ClusterConfig, joinToken, spireHost, ppsHost, knoxGateway, spireTrustBundle string, enableLogs bool) *InitConfig {
	return &InitConfig{
		ClusterConfig: cc,
		JoinToken:     joinToken,
		SpireHost:     spireHost,
		PPSHost:       ppsHost,
		KnoxGateway:   knoxGateway,

		SpireTrustBundleURL: spireTrustBundle,
		EnableLogs:          enableLogs,
	}
}
func (ic *InitConfig) CreateBaseTemplateConfig() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	spireHost, spirePort, err := parseURL(ic.SpireHost)
	if err != nil {
		return err
	}
	if spirePort == "80" {
		// default spire port
		spirePort = "8081"
	}

	// currently unused as we use insecure bootstrap
	var spireTrustBundleURL = ic.SpireTrustBundleURL
	if spireTrustBundleURL == "" {
		if strings.Contains(ic.SpireHost, "spire.dev.accuknox.com") {
			spireTrustBundleURL = spireTrustBundleURLMap["dev"]
		} else if strings.Contains(ic.SpireHost, "spire.stage.accuknox.com") {
			spireTrustBundleURL = spireTrustBundleURLMap["stage"]
		} else if strings.Contains(ic.SpireHost, "spire.demo.accuknox.com") {
			spireTrustBundleURL = spireTrustBundleURLMap["demo"]
		} else if strings.Contains(ic.SpireHost, "spire.prod.accuknox.com") {
			spireTrustBundleURL = spireTrustBundleURLMap["prod"]
		} else if strings.Contains(ic.SpireHost, "spire.xcitium.accuknox.com") {
			spireTrustBundleURL = spireTrustBundleURLMap["xcitium"]
		}
	}
	ic.TCArgs = TemplateConfigArgs{
		ReleaseVersion: ic.AgentsVersion,

		KubeArmorImage:            ic.KubeArmorImage,
		KubeArmorInitImage:        ic.KubeArmorInitImage,
		KubeArmorRelayServerImage: ic.KubeArmorRelayServerImage,
		KubeArmorVMAdapterImage:   ic.KubeArmorVMAdapterImage,
		SPIREAgentImage:           ic.SPIREAgentImage,
		SIAImage:                  ic.SIAImage,
		PEAImage:                  ic.PEAImage,
		FeederImage:               ic.FeederImage,
		DiscoverImage:             ic.DiscoverImage,
		SumEngineImage:            ic.SumEngineImage,
		HardeningAgentImage:       ic.HardeningAgentImage,

		Hostname: hostname,
		// TODO: make configurable
		KubeArmorURL:  "kubearmor:32767",
		KubeArmorPort: "32767",

		RelayServerURL:  "kubearmor-relay-server:32768",
		RelayServerAddr: "kubearmor-relay-server",
		RelayServerPort: "32768",

		WorkerNode: ic.WorkerNode,

		SIAAddr:    "shared-informer-agent:32769",
		PEAAddr:    "policy-enforcement-agent:32770",
		HardenAddr: "hardening-agent:32771",

		EnableLogs: ic.EnableLogs,

		PPSHost: ic.PPSHost,

		JoinToken:     ic.JoinToken,
		SpireHostAddr: spireHost,
		SpireHostPort: spirePort,

		SpireTrustBundleURL: spireTrustBundleURL,

		// kubearmor config
		KubeArmorVisibility:     ic.Visibility,
		KubeArmorHostVisibility: ic.HostVisibility,

		KubeArmorFilePosture:    ic.DefaultFilePosture,
		KubeArmorNetworkPosture: ic.DefaultNetworkPosture,
		KubeArmorCapPosture:     ic.DefaultCapPosture,

		KubeArmorHostFilePosture:    ic.DefaultHostFilePosture,
		KubeArmorHostNetworkPosture: ic.DefaultHostNetworkPosture,
		KubeArmorHostCapPosture:     ic.DefaultHostCapPosture,

		NetworkCIDR: ic.CIDR,

		SecureContainers: ic.SecureContainers,

		VmMode: ic.Mode,
	}
	return nil
}

func (ic *InitConfig) InitializeControlPlane() error {
	// validate this environment
	dockerStatus, err := ic.ValidateEnv()
	if err != nil {
		return err
	}
	fmt.Println(dockerStatus)

	configPath, err := createDefaultConfigPath()
	if err != nil {
		return err
	}
	ic.TCArgs.KubeArmorImage = ic.KubeArmorImage
	ic.TCArgs.KubeArmorInitImage = ic.KubeArmorInitImage
	ic.TCArgs.KubeArmorRelayServerImage = ic.KubeArmorRelayServerImage
	ic.TCArgs.KubeArmorVMAdapterImage = ic.KubeArmorVMAdapterImage

	// agents
	ic.TCArgs.SIAImage = ic.SIAImage
	ic.TCArgs.PEAImage = ic.PEAImage
	ic.TCArgs.FeederImage = ic.FeederImage
	ic.TCArgs.DiscoverImage = ic.DiscoverImage
	ic.TCArgs.SumEngineImage = ic.SumEngineImage
	ic.TCArgs.HardeningAgentImage = ic.HardeningAgentImage

	ic.TCArgs.KubeArmorURL = "kubearmor:32767"
	ic.TCArgs.KubeArmorPort = "32767"

	ic.TCArgs.RelayServerURL = "kubearmor-relay-server:32768"
	ic.TCArgs.RelayServerAddr = "kubearmor-relay-server"
	ic.TCArgs.RelayServerPort = "32768"

	ic.TCArgs.WorkerNode = ic.WorkerNode

	ic.TCArgs.SIAAddr = "shared-informer-agent:32769"
	ic.TCArgs.PEAAddr = "policy-enforcement-agent:32770"
	ic.TCArgs.HardenAddr = "hardening-agent:32771"
	ic.TCArgs.ImagePullPolicy = string(ic.ImagePullPolicy)

	ic.TCArgs.ConfigPath = configPath

	ic.TCArgs.DiscoverRules = combineVisibilities(ic.Visibility, ic.HostVisibility)
	ic.TCArgs.ProcessOperation = isOperationDisabled(ic.Visibility, ic.HostVisibility, "process")
	ic.TCArgs.FileOperation = isOperationDisabled(ic.Visibility, ic.HostVisibility, "file")
	ic.TCArgs.NetworkOperation = isOperationDisabled(ic.Visibility, ic.HostVisibility, "network")

	// initialize sprig for templating
	sprigFuncs := sprig.GenericFuncMap()

	// write compose file
	composeFilePath, err := copyOrGenerateFile(ic.UserConfigPath, configPath, "docker-compose.yaml", sprigFuncs, cpComposeFileTemplate, ic.TCArgs)
	if err != nil {
		return err
	}

	// List of config files to be generated or copied
	// TODO: Refactor later
	agentMeta := getAgentConfigMeta()

	kmuxConfigArgs := KmuxConfigTemplateArgs{
		ReleaseVersion: ic.AgentsVersion,
		StreamName:     "knox-gateway",
		ServerURL:      ic.KnoxGateway,
		RMQServer:      "rabbitmq:5672",
	}

	// Generate or copy config files
	for _, agentObj := range agentMeta {
		tcArgs := ic.TCArgs
		tcArgs.KmuxConfigPath = agentObj.kmuxConfigPath

		// generate config file if not empty
		if agentObj.configPath != "" {
			if _, err := copyOrGenerateFile(ic.UserConfigPath, configPath, agentObj.configPath, sprigFuncs, agentObj.configTemplateString, tcArgs); err != nil {
				return err
			}
		}

		// generate kmux config only if it exists for this agent
		if agentObj.kmuxConfigPath != "" {
			if _, err := copyOrGenerateFile(ic.UserConfigPath, configPath, agentObj.kmuxConfigPath, sprigFuncs, agentObj.kmuxConfigTemplateString, kmuxConfigArgs); err != nil {
				return err
			}
		}
	}

	// Diagnose if necessary and run compose command
	return ic.runComposeCommand(composeFilePath)
}

// runComposeCommand runs the Docker Compose command with the necessary arguments
func (ic *InitConfig) runComposeCommand(composeFilePath string) error {
	diagnosis := true
	args := []string{
		"-f", composeFilePath, "--profile", "spire-agent",
		"--profile", "kubearmor", "--profile", "accuknox-agents",
		"up", "-d",
	}

	if semver.Compare(ic.composeVersion, common.MinDockerComposeWithWaitSupported) >= 0 {
		args = append(args, "--wait", "--wait-timeout", "60")
	} else {
		diagnosis = false
	}

	// run compose command
	_, err := ExecComposeCommand(true, ic.DryRun, ic.composeCmd, args...)
	if err != nil {
		// cleanup volumes
		_, volDelErr := ExecDockerCommand(true, false, "docker", "volume", "rm", "spire-vol", "kubearmor-init-vol")
		if volDelErr != nil {
			fmt.Println("Error while removing volumes:", volDelErr.Error())
		}
		return ic.handleComposeError(err, diagnosis)
	}
	return nil
}

// handleComposeError handles errors from the Docker Compose command
func (ic *InitConfig) handleComposeError(err error, diagnosis bool) error {
	if diagnosis {
		diagnosisResult, diagErr := diagnose(NodeType_ControlPlane)
		if diagErr != nil {
			diagnosisResult = diagErr.Error()
		}
		return fmt.Errorf("Error: %s.\n\nDIAGNOSIS:\n%s", err.Error(), diagnosisResult)
	}
	return err
}

func combineVisibilities(visibility, hostVisibility string) string {
	visibilities := make(map[string]struct{})
	for _, vis := range strings.Split(visibility+","+hostVisibility, ",") {
		visibilities[vis] = struct{}{}
	}

	combined := make([]string, 0, len(visibilities))
	for vis := range visibilities {
		combined = append(combined, vis)
	}

	return strings.Join(combined, ",")
}

// isOperationDisabled returns true if the operation is not included in the combined visibility settings.
func isOperationDisabled(visibility, hostVisibility, operation string) bool {
	visibilities := make(map[string]struct{})
	for _, vis := range strings.Split(visibility+","+hostVisibility, ",") {
		visibilities[vis] = struct{}{}
	}
	_, exists := visibilities[operation]
	return !exists
}

func getAgentConfigMeta() []agentConfigMeta {
	agentMeta := []agentConfigMeta{
		{
			agentName:            "spire",
			configPath:           "spire/conf/agent.conf",
			configTemplateString: spireAgentConfig,
		},
		{
			agentName:                "sia",
			configPath:               "sia/app.yaml",
			configTemplateString:     siaConfig,
			kmuxConfigPath:           "/opt/sia/kmux-config.yaml",
			kmuxConfigTemplateString: kmuxConfig,
		},
		{
			agentName:                "pea",
			configPath:               "pea/application.yaml",
			configTemplateString:     peaConfig,
			kmuxConfigPath:           "/opt/pea/kmux-config.yaml",
			kmuxConfigTemplateString: kmuxConfig},
		{
			agentName:                "feeder-service",
			kmuxConfigPath:           "/opt/feeder-service/kmux-config.yaml",
			kmuxConfigTemplateString: kmuxConfig,
		},
		{
			agentName:                "sumengine",
			configPath:               "sumengine/config.yaml",
			configTemplateString:     sumEngineConfig,
			kmuxConfigPath:           "/opt/sumengine/kmux-config.yaml",
			kmuxConfigTemplateString: sumEngineKmuxConfig,
		},
		{
			agentName:                "discover",
			configPath:               "discover/config.yaml",
			configTemplateString:     discoverConfig,
			kmuxConfigPath:           "/opt/discover/kmux-config.yaml",
			kmuxConfigTemplateString: discoverKmuxConfig,
		},
		{
			agentName:                "hardening-agent",
			configPath:               "hardening-agent/config.yaml",
			configTemplateString:     hardeningAgentConfig,
			kmuxConfigPath:           "/opt/hardening-agent/kmux-config.yaml",
			kmuxConfigTemplateString: hardeningAgentKmuxConfig,
		},
	}

	return agentMeta
}
