/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// exec2Cmd represents the exec2 command
var exec2Cmd = &cobra.Command{
	Use:   "exec2",
	Short: "A brief description of your command",
	Long:  `This will keep the replicaset constant and install kubearmor.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("exec2 called")
	},
}

func init() {
	rootCmd.AddCommand(exec2Cmd)
}
