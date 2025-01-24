package cmd

import (
	"github.com/accuknox/accuknox-cli-v2/pkg/api/asset"
	"github.com/spf13/cobra"
)

// assetCmd represents the parent command for assets
var assetCmd = &cobra.Command{
	Use:     "asset",
	Short:   "Managing assets, with subcommands like `list`",
	Long:    "Managing assets, with subcommands like `list`",
	Example: asset.AssetDescription,
}

func init() {
	apiCmd.AddCommand(assetCmd)
}
