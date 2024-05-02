package cmd

import (
	"fmt"
	"os"

	"github.com/accuknox/accuknox-cli-v2/pkg/deboard"
	"github.com/accuknox/accuknox-cli-v2/pkg/onboard"
	"github.com/spf13/cobra"
)

// deboardNodeCmd represents the deboardNode command
var deboardNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Deboard a worker node",
	Long:  "Deboard a worker node",
	RunE: func(cmd *cobra.Command, args []string) error {

		if vmMode == "" {
			systemdInstallation, err := onboard.CheckSystemdInstallation()
			if err != nil {
				return fmt.Errorf("error checking systemd files")
			}
			if systemdInstallation {
				vmMode = onboard.VMMode_Systemd
			} else {
				vmMode = onboard.VMMode_Docker
			}

		}

		switch vmMode {
		case onboard.VMMode_Systemd:
			_, err := deboard.Deboard(onboard.NodeType_WorkerNode, vmMode, dryRun)
			if err != nil {
				return fmt.Errorf("Failed to deboard worker node: %s", err.Error())
			}
			fmt.Println("Worker node deboarded successfully.")
			return nil
		case onboard.VMMode_Docker:
			configPath, err := deboard.Deboard(onboard.NodeType_WorkerNode, vmMode, dryRun)
			if err != nil && os.IsPermission(err) {
				fmt.Println("Please remove any remaining resources at", configPath)
			} else if err != nil {
				return fmt.Errorf("Failed to deboard worker node: %s", err.Error())
			}
			fmt.Println("Worker node deboarded successfully.")
			return nil

		default:
			fmt.Printf("vm mode: %s invalid, accepted values (docker/systemd)", vmMode)
		}
		return nil
	},
}

func init() {
	deboardVMCmd.AddCommand(deboardNodeCmd)
}
