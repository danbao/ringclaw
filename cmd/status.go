package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if weclaw is running in background",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := readPid()
		if err != nil {
			fmt.Println("weclaw is not running")
			return nil
		}

		if processExists(pid) {
			fmt.Printf("weclaw is running (pid=%d)\n", pid)
			fmt.Printf("Log: %s\n", logFile())
		} else {
			fmt.Println("weclaw is not running (stale pid file)")
		}
		return nil
	},
}
