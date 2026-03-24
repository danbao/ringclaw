package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background weclaw process",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := readPid()
		if err != nil {
			fmt.Println("weclaw is not running")
			return nil
		}

		if !processExists(pid) {
			os.Remove(pidFile())
			fmt.Println("weclaw is not running (stale pid file removed)")
			return nil
		}

		p, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find process: %w", err)
		}

		if err := p.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("stop process: %w", err)
		}

		os.Remove(pidFile())
		fmt.Printf("weclaw stopped (pid=%d)\n", pid)
		return nil
	},
}
