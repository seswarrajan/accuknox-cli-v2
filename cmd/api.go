package cmd

import (
	"github.com/accuknox/accuknox-cli-v2/pkg/api/asset"
	"github.com/accuknox/accuknox-cli-v2/pkg/api/cluster"
	"github.com/accuknox/accuknox-cli-v2/pkg/config"
	"github.com/spf13/cobra"
)

var (
	CWPP_URL  string
	CSPM_URL  string
	TOKEN     string
	TENANT_ID string
	CFG_FILE  string
)

// apiCmd represents the root API command
var apiCmd = &cobra.Command{
	Use:     "api",
	Short:   "API-related commands",
	Long:    "Root command for all API related subcommands.",
	Example: asset.AssetDescription + cluster.ClusterListDescription + cluster.ClusterAlertDescription + cluster.ClusterPolicyDescription,
}

func init() {
	apiCmd.PersistentFlags().StringVar(&CWPP_URL, "cwpp_url", config.Cfg.CWPP_URL, "CWPP url")
	apiCmd.PersistentFlags().StringVar(&CSPM_URL, "cspm_url", "", "Set CSPM URL")
	apiCmd.PersistentFlags().StringVar(&TOKEN, "token", "", "Set Token")
	apiCmd.PersistentFlags().StringVar(&TENANT_ID, "tenant-id", "", "Set Tenant-id")
	apiCmd.PersistentFlags().StringVar(&CFG_FILE, "cfgFile", "$HOME/.accuknox.cfg", "Set Config File")

	rootCmd.AddCommand(apiCmd)
}
