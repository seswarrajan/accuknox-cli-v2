package cmd

import (
	"os"

	"github.com/accuknox/accuknox-cli-v2/pkg/deboard"
	"github.com/accuknox/accuknox-cli-v2/pkg/logger"
	"github.com/spf13/cobra"
)

// deboardNodeCmd represents the deboardNode command
var deboardRATScanner = &cobra.Command{
	Use:   "scanner",
	Short: "Deboard RAT scanner",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := deboard.UninstallRAT()
		if err != nil {
			if os.IsNotExist(err) {
				logger.Info1("RAT Installation not found")
			} else {
				logger.Error("error removing RAT installation:%s", err.Error())
				return err
			}
		} else {
			logger.PrintSuccess("RAT uninstalled successfully.")
		}
		return nil
	},
}

func init() {
	deboardVMCmd.AddCommand(deboardRATScanner)
}
