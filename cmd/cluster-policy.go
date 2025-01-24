package cmd

import (
	"github.com/accuknox/accuknox-cli-v2/pkg/api/cluster"
	"github.com/accuknox/accuknox-cli-v2/pkg/config"
	"github.com/accuknox/accuknox-cli-v2/pkg/logger"
	"github.com/spf13/cobra"
)

var clusterPolicyOptions cluster.ClusterPolicyOptions

var clusterPolicyCmd = &cobra.Command{
	Use:     "policy",
	Short:   "Enlist the cluster policies. These include all policies, including, KubeArmor, Network, Admission Controller policies",
	Long:    `Enlist the cluster policies. These include all policies, including, KubeArmor, Network, Admission Controller policies.`,
	Example: cluster.ClusterPolicyDescription,
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.LoadConfig(CFG_FILE); err != nil {
			logger.Error(err.Error())
			return
		}
		config.SetConfig(CWPP_URL, CSPM_URL, TOKEN, TENANT_ID)
		cluster.FetchAndProcessPolicies(clusterPolicyOptions)
	},
}

func init() {
	clusterCmd.AddCommand(clusterPolicyCmd)

	clusterPolicyCmd.Flags().StringVar(&clusterPolicyOptions.ClusterListJQ, "clusterjq", ".[]", "JQ filter to apply on cluster list output")
	clusterPolicyCmd.Flags().StringVar(&clusterPolicyOptions.PolicyJQ, "policyjq", ".list_of_policies[]", "JQ filter for policy")
	clusterPolicyCmd.Flags().StringVar(&clusterPolicyOptions.Operation, "operation", "list", "operation")
	clusterPolicyCmd.Flags().StringVar(&clusterPolicyOptions.ClusterName, "clusterName", "", "list policy for given cluster name")
	clusterPolicyCmd.Flags().BoolVar(&clusterPolicyOptions.JsonFormat, "json", false, "print policies in json format")
}
