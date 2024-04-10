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

	// initialize sprig for templating
	sprigFuncs := sprig.GenericFuncMap()

	_, err = copyOrGenerateFile("", cm.KAconfigPath, "kubearmor.yaml", sprigFuncs, kubeArmorConfig, ic.TCArgs)
	if err != nil {
		return err
	}
	_, err = copyOrGenerateFile("", cm.VmAdapterconfigPath, "vm-adapter-config.yaml", sprigFuncs, vmAdapterConfig, ic.TCArgs)
	if err != nil {
		return err
	}

	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.PEAconfigPath, "conf/application.yaml", sprigFuncs, peaConfig, ic.TCArgs)
	if err != nil {
		return err
	}
	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.SIAconfigPath, "conf/app.yaml", sprigFuncs, siaConfig, ic.TCArgs)
	if err != nil {
		return err
	}

	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.SpireconfigPath, "conf/agent/agent.conf", sprigFuncs, spireAgentConfig, ic.TCArgs)
	if err != nil {
		return err
	}

	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.FSconfigPath, "conf/env", sprigFuncs, fsEnvVal, ic.TCArgs)
	if err != nil {
		return err
	}

	kmuxConfigArgs := KmuxConfigTemplateArgs{
		ReleaseVersion: ic.AgentsVersion,
		StreamName:     "knox-gateway",
		ServerURL:      ic.KnoxGateway,
	}

	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.SIAconfigPath, "kmux-config.yaml", sprigFuncs, kmuxConfig, kmuxConfigArgs)
	if err != nil {
		return err
	}

	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.FSconfigPath, "kmux-config.yaml", sprigFuncs, kmuxConfig, kmuxConfigArgs)
	if err != nil {
		return err
	}
	_, err = copyOrGenerateFile(ic.UserConfigPath, cm.PEAconfigPath, "kmux-config.yaml", sprigFuncs, kmuxConfig, kmuxConfigArgs)
	if err != nil {
		return err
	}

	services := []string{"spire-agent.service", "kubearmor.service", "kubearmor-relay-server.service", "kubearmor-vm-adapter.service", "accuknox-policy-enforcement-agent.service", "accuknox-shared-informer-agent.service", "accuknox-feeder-service.service"}

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

	_, err = copyOrGenerateFile("", cm.SystemdDir, cm.Feeder_service+".service", sprigFuncs, feederServiceFile, interface{}(nil))
	if err != nil {
		return err
	}
	_, err = copyOrGenerateFile("", cm.SystemdDir, cm.Pea_agent+".service", sprigFuncs, peaServiceFile, interface{}(nil))
	if err != nil {
		return err
	}
	_, err = copyOrGenerateFile("", cm.SystemdDir, cm.Sia_agent+".service", sprigFuncs, siaServiceFile, interface{}(nil))
	if err != nil {
		return err
	}
	_, err = copyOrGenerateFile("", cm.SystemdDir, cm.Relay_server+".service", sprigFuncs, relayServerServiceFile, interface{}(nil))
	if err != nil {
		return err
	}
	return nil
}
