package onboard

import (
	"fmt"

	"github.com/Masterminds/sprig"
	cm "github.com/accuknox/accuknox-cli-v2/pkg/common"
)

func (ic *InitConfig) InitializeControlPlaneSD() error {

	ic.TCArgs.KubeArmorURL = "0.0.0.0:32767"
	ic.TCArgs.KubeArmorPort = "32767"

	ic.TCArgs.RelayServerURL = "0.0.0.0:32768"
	ic.TCArgs.RelayServerAddr = "0.0.0.0"
	ic.TCArgs.RelayServerPort = "32768"

	ic.TCArgs.WorkerNode = ic.WorkerNode

	ic.TCArgs.SIAAddr = "0.0.0.0:32769"
	ic.TCArgs.PEAAddr = "0.0.0.0:32770"
	ic.TCArgs.VmMode = ic.Mode

	err := SystemdInstall(ic.ClusterConfig)
	if err != nil {
		fmt.Println(cm.Red + "Installation failed!! Cleaning up downloaded assets:" + cm.Reset)
		// ignoring G104 - can't send nil in installation failed case
		Deletedir(cm.Download_dir)
		DeboardSystemd(NodeType_ControlPlane) // #nosec G104
		return err
	}
	ic.TCArgs.KmuxConfigPathFS = "/opt/accuknox-feeder-service/kmux-config.yaml"
	ic.TCArgs.KmuxConfigPathSIA = "/opt/accuknox-shared-informer-agent/kmux-config.yaml"
	ic.TCArgs.KmuxConfigPathPEA = "/opt/accuknox-policy-enforcement-agent/kmux-config.yaml"
	ic.TCArgs.KmuxConfigPathSumengine = "/opt/accuknox-sumengine/kmux-config.yaml"
	ic.TCArgs.KmuxConfigPathDiscover = "/opt/accuknox-discover/kmux-config.yaml"

	// initialize sprig for templating
	sprigFuncs := sprig.GenericFuncMap()

	configs := []struct {
		userConfigDir  string
		dirPath        string
		filePath       string
		templateString string
	}{
		{"", cm.KAconfigPath, "kubearmor.yaml", kubeArmorConfig},
		{"", cm.VmAdapterconfigPath, "vm-adapter-config.yaml", vmAdapterConfig},
		{ic.UserConfigPath, cm.PEAconfigPath, "conf/application.yaml", peaConfig},
		{ic.UserConfigPath, cm.SIAconfigPath, "conf/app.yaml", siaConfig},
		{ic.UserConfigPath, cm.SpireconfigPath, "conf/agent/agent.conf", spireAgentConfig},
		{ic.UserConfigPath, cm.FSconfigPath, "conf/env", fsEnvVal},
		{ic.UserConfigPath, cm.DiscoverConfigPath, "conf/config.yaml", discoverConfig},
		{ic.UserConfigPath, cm.SumEngineConfigPath, "conf/config.yaml", sumEngineConfig},
	}

	for _, cfg := range configs {
		_, err = copyOrGenerateFile(cfg.userConfigDir, cfg.dirPath, cfg.filePath, sprigFuncs, cfg.templateString, ic.TCArgs)
		if err != nil {
			return err
		}
	}

	kmuxConfigArgs := KmuxConfigTemplateArgs{
		ReleaseVersion: ic.AgentsVersion,
		StreamName:     "knox-gateway",
		ServerURL:      ic.KnoxGateway,
		RMQServer:      "0.0.0.0:5672",
	}

	dirPathTemplateMap := map[string]string{
		cm.SIAconfigPath:       kmuxConfig,
		cm.FSconfigPath:        kmuxConfig,
		cm.PEAconfigPath:       kmuxConfig,
		cm.SumEngineConfigPath: sumEngineKmuxConfig,
		cm.DiscoverConfigPath:  discoverKmuxConfig,
	}

	for dirPath, templateString := range dirPathTemplateMap {
		_, err = copyOrGenerateFile(ic.UserConfigPath, dirPath, "kmux-config.yaml", sprigFuncs, templateString, kmuxConfigArgs)
		if err != nil {
			return err
		}
	}

	services := []string{"spire-agent.service", "kubearmor.service", "kubearmor-relay-server.service", "kubearmor-vm-adapter.service", "accuknox-policy-enforcement-agent.service", "accuknox-shared-informer-agent.service", "accuknox-feeder-service.service", "accuknox-sumengine.service", "accuknox-discover.service"}

	for _, serviceName := range services {
		err = StartSystemdService(serviceName)
		if err != nil {
			fmt.Printf("failed to start service %s: %s\n", serviceName, err.Error())
			return err
		}
	}

	// Clean Up
	fmt.Println("Cleaning up downloaded assets")
	Deletedir(cm.Download_dir)
	return nil

}
func placeServiceFiles(workernode bool) error {
	sprigFuncs := sprig.GenericFuncMap()

	_, err := copyOrGenerateFile("", cm.SystemdDir, cm.Vm_adapter+".service", sprigFuncs, vmAdapterServiceFile, interface{}(nil))
	if err != nil {
		return err
	}

	if workernode {
		return nil
	}

	filePathTemplateMap := map[string]string{
		cm.Feeder_service + ".service": feederServiceFile,
		cm.Pea_agent + ".service":      peaServiceFile,
		cm.Sia_agent + ".service":      siaServiceFile,
		cm.Relay_server + ".service":   relayServerServiceFile,
		cm.Summary_Engine + ".service": sumEngineFile,
		cm.Discover_Agent + ".service": discoverFile,
	}

	for filePath, templateString := range filePathTemplateMap {
		_, err = copyOrGenerateFile("", cm.SystemdDir, filePath, sprigFuncs, templateString, interface{}(nil))
		if err != nil {
			return err
		}
	}

	return nil
}
