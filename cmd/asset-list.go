package cmd

import (
	"github.com/accuknox/accuknox-cli-v2/pkg/api/asset"
	"github.com/accuknox/accuknox-cli-v2/pkg/config"
	"github.com/accuknox/accuknox-cli-v2/pkg/logger"
	"github.com/spf13/cobra"
)

var assetOptions asset.Options

// assetListCmd represents the `list` subcommand
var assetListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List assets",
	Long:    `List the assets available with optional filtering using flags.`,
	Example: asset.AssetDescription,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadConfig(CFG_FILE); err != nil {
			logger.Error(err.Error())
			return
		}
		config.SetConfig(CWPP_URL, CSPM_URL, TOKEN, TENANT_ID)
		asset.ListAssets(assetOptions)
	},
}

func init() {
	assetCmd.AddCommand(assetListCmd)

	assetListCmd.Flags().StringVarP(&assetOptions.Filter, "filter", "f", "", "Category filter to be used on asset list (e.g., asset_category=Container)")
	assetListCmd.Flags().StringVar(&assetOptions.AssetJQ, "assetjq", ".results[]", "jq-based filtering to use on asset list")
	assetListCmd.Flags().IntVar(&assetOptions.Page, "page", 0, "Page number for asset listing")
	assetListCmd.Flags().IntVar(&assetOptions.PageSize, "page-size", 50, "Number of assets to list per page")
	assetListCmd.Flags().BoolVar(&assetOptions.NoPager, "noPager", false, "Dumps complete list without pagination")
	assetListCmd.Flags().IntVarP(&assetOptions.Timeout, "timeout", "t", 60, "Set timeout in secs")
	assetListCmd.Flags().BoolVar(&assetOptions.JsonFormat, "json", false, "List assets in the JSON format")
}
