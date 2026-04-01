package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	fileCmd.AddCommand(fileUploadCmd)
	rootCmd.AddCommand(fileCmd)
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "File operations",
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload <chatId> <file-path>",
	Short: "Upload a local file to a chat",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		data, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		fileName := filepath.Base(args[1])
		resp, err := client.UploadFile(ctx, args[0], fileName, data)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
		if jsonOutput {
			printJSON(resp)
		} else {
			fmt.Printf("File uploaded: %s — %s\n", resp.ID, resp.Name)
		}
		return nil
	},
}
