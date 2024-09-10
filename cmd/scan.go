package cmd

import (
	"fmt"

	"github.com/accuknox/accuknox-cli-v2/pkg/scan"
	"github.com/spf13/cobra"
)

var scanOpts scan.ScanOptions

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Runtime scanning of CI/CD pipelines",
	Long:  "Scans the events taking place in CI/CD pipelines powered by KubeArmor",
	RunE: func(cmd *cobra.Command, args []string) error {
		scanner := scan.New(&scanOpts)

		if err := scanner.Start(); err != nil {
			fmt.Println(err)
			return err
		}

		return nil
	},
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policies for scanning",
	Long:  "Apply, generate, or manage policies for the scanning process",
	RunE: func(cmd *cobra.Command, args []string) error {
		if scanOpts.PolicyAction != "Block" && scanOpts.PolicyAction != "Audit" {
			return fmt.Errorf("invalid policy action: %s. Must be 'Block' or 'Audit'", scanOpts.PolicyAction)
		}

		if scanOpts.PolicyEvent != "ADDED" && scanOpts.PolicyEvent != "DELETED" {
			return fmt.Errorf("invalid policy event: %s. Must be 'ADDED' or 'DELETED'", scanOpts.PolicyEvent)
		}

		scanner := scan.New(&scanOpts)
		return scanner.HandlePolicies()
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.AddCommand(policyCmd)

	scanCmd.PersistentFlags().BoolVar(&scanOpts.FilterEventType.All, "all", false, "Collect 'all' events, may get verbose")
	scanCmd.PersistentFlags().BoolVar(&scanOpts.FilterEventType.System, "system", false, "Collect 'system' only events")
	scanCmd.PersistentFlags().BoolVar(&scanOpts.AlertFilters.DetailedView, "detailed-view", false, "Detailed view contains raw JSON and complete policy applied")
	scanCmd.PersistentFlags().StringVar(&scanOpts.AlertFilters.IgnoreEvent, "ignore-alerts", "", "Ignore alerts of a specific type: 'file', 'network', or 'process'")
	scanCmd.PersistentFlags().StringVar(&scanOpts.AlertFilters.SeverityLevel, "min-severity", "", "Minimum severity level for alerts (1-10)")

	policyCmd.Flags().BoolVar(&scanOpts.PolicyDryRun, "dryrun", false, "Generate and save the hardening policies but don't apply them")
	policyCmd.Flags().BoolVar(&scanOpts.StrictMode, "strict", false, "In strict mode all the policies will be applied, this may lead to a lot of alerts generated")
	policyCmd.Flags().StringVar(&scanOpts.PolicyAction, "action", "Audit", "Policy action: 'Block' or 'Audit'")
	policyCmd.Flags().StringVar(&scanOpts.PolicyEvent, "event", "ADDED", "Policy event: 'ADDED' or 'DELETED'")
	policyCmd.Flags().StringVar(&scanOpts.PoliciesPath, "policies", "", "File path to user defined security policies to be applied")
}
