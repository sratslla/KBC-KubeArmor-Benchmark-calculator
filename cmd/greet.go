package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// greetCmd represents the greet command
var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Greets the people of KubeArmor",
	Long:  `This command prints a greeting message: "Hello, people of KubeArmor"`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, people of KubeArmor")
	},
}

func init() {
	rootCmd.AddCommand(greetCmd)
}
