package onboard

import (
	"fmt"

	"github.com/Masterminds/sprig"
	cm "github.com/accuknox/accuknox-cli-v2/pkg/common"
)

func (jc *JoinConfig) JoinSystemdNode() error {

	// Install packages

	err := SystemdInstall(jc.ClusterConfig)
	if err != nil {
		fmt.Println("Installation failed!! Cleaning up downloaded assets", err)
		Deletedir(cm.Download_dir)
		DeboardSystemd(NodeType_WorkerNode) // #nosec G104
		return err
	}
	sprigFuncs := sprig.GenericFuncMap()

	// write config file and copy it to /opt/kubearmor/kubearmor
	_, err = copyOrGenerateFile("", cm.KAconfigPath, "kubearmor.yaml", sprigFuncs, kubeArmorConfig, jc.TCArgs)
	if err != nil {
		return err
	}

	_, err = copyOrGenerateFile("", cm.VmAdapterconfigPath, "vm-adapter-config.yaml", sprigFuncs, vmAdapterConfig, jc.TCArgs)
	if err != nil {
		return err
	}
	// start kubearmor
	err = StartSystemdService("kubearmor.service")
	if err != nil {
		fmt.Println("Failed to start kubearmor.service", err)
	}
	err = StartSystemdService("kubearmor-vm-adapter.service")
	if err != nil {
		fmt.Println("Failed to start kubearmor-vm-adapter.service", err)
	}
	fmt.Println("Cleaning up downloaded assets")
	Deletedir(cm.Download_dir)
	return nil
}
func SystemdInstall(cc ClusterConfig) error {
	// initialize sprig for templating

	// Installing KubeArmor

	fmt.Println("Installing Kubearmor...")
	err := InstallKa(cc.KubeArmorTag)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Kubearmor downloaded successfully")

	//install vm-adapter
	fmt.Println("Installing kubearmor-vm-adapter...")
	err = InstallAgent(cm.Vm_adapter, cc.VmAdapterTag)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("vm-adapter version: %s downloaded successfully\n", cc.VmAdapterTag)

	if !cc.WorkerNode {
		// Install other agents on cp node
		agents := []struct {
			name    string
			version string
		}{

			{cm.Spire_agent, cc.SpireTag},
			{cm.Relay_server, cc.RelayServerTag},
			{cm.Sia_agent, cc.SiaTag},
			{cm.Pea_agent, cc.PeaTag},
			{cm.Feeder_service, cc.FsTag},
			{cm.Summary_Engine, cc.SumEngineTag},
			{cm.Discover_Agent, cc.DiscoverTag},
		}

		for _, agent := range agents {
			fmt.Printf("Installing %s...\n", agent.name)
			err := InstallAgent(agent.name, agent.version)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Printf("%s version %s downloaded successfully\n", agent.name, agent.version)
		}

	}
	err = placeServiceFiles(cc.WorkerNode)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("All agents downloaded successfully")

	return nil
}
