// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of KubeArmor

package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// HTTPIP : IP of the http request
	HTTPIP string
	// HTTPPort : Port of the http request
	HTTPPort string
)

// vmCmd represents the vm command
var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Commands for post-installation VM operations",
	Long:  "Commands for post-installation VM operations",
}

// ========== //
// == Init == //
// ========== //

func init() {
	rootCmd.AddCommand(vmCmd)

	// options for vm generic commands related to HTTP Request
	vmCmd.PersistentFlags().StringVar(&HTTPIP, "http-ip", "127.0.0.1", "IP of KubeArmor")
	vmCmd.PersistentFlags().StringVar(&HTTPPort, "http-port", "8000", "Port of KubeArmor")
}
