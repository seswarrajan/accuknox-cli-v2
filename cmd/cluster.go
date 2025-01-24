package cmd

import (
	"github.com/accuknox/accuknox-cli-v2/pkg/api/cluster"
	"github.com/spf13/cobra"
)

// clusterCmd represents the parent command for clusters
var clusterCmd = &cobra.Command{
	Use:     "cluster",
	Short:   "Provides cluster info, like listing clusters, nodes, alerts, policies",
	Long:    `Provides cluster info, like listing clusters, nodes, alerts, policies`,
	Example: cluster.ClusterListDescription + cluster.ClusterAlertDescription + cluster.ClusterPolicyDescription,
}

func init() {
	apiCmd.AddCommand(clusterCmd)
}
