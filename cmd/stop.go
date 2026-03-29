package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background ringclaw process",
	RunE: func(cmd *cobra.Command, args []string) error {
		stopAllRingclaw()
		fmt.Println("ringclaw stopped")
		return nil
	},
}
