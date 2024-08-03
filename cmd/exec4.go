/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// exec4Cmd represents the exec4 command
var exec4Cmd = &cobra.Command{
	Use:   "exec4",
	Short: "A brief description of your command",
	Long:  `exec4`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("exec4 called")
		time.Sleep(3 * time.Minute)
		calculateBenchMark()
	},
}

func init() {
	rootCmd.AddCommand(exec4Cmd)
}
