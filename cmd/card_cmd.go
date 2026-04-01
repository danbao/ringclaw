package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	cardCmd.AddCommand(cardCreateCmd)
	cardCmd.AddCommand(cardGetCmd)
	cardCmd.AddCommand(cardDeleteCmd)
	rootCmd.AddCommand(cardCmd)
}

var cardCmd = &cobra.Command{
	Use:   "card",
	Short: "Adaptive card operations",
}

var cardCreateCmd = &cobra.Command{
	Use:     "create <chatId> <json-file>",
	Short:   "Create an adaptive card from a JSON file",
	Example: "  ringclaw card create <chatId> card.json",
	Args:    cobra.ExactArgs(2),
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
		if !json.Valid(data) {
			return fmt.Errorf("invalid JSON in %s", args[1])
		}

		card, err := client.CreateAdaptiveCard(ctx, args[0], json.RawMessage(data))
		if err != nil {
			return fmt.Errorf("create card failed: %w", err)
		}
		if jsonOutput {
			printJSON(card)
		} else {
			fmt.Printf("Card created: %s\n", card.ID)
		}
		return nil
	},
}

var cardGetCmd = &cobra.Command{
	Use:   "get <cardId>",
	Short: "Get adaptive card details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		card, err := client.GetAdaptiveCard(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get card failed: %w", err)
		}
		if jsonOutput {
			printJSON(card)
		} else {
			fmt.Printf("Card: %s\n", card.ID)
			fmt.Printf("  Type:    %s\n", card.Type)
			fmt.Printf("  Version: %s\n", card.Version)
			fmt.Printf("  Created: %s\n", card.CreationTime)
		}
		return nil
	},
}

var cardDeleteCmd = &cobra.Command{
	Use:   "delete <cardId>",
	Short: "Delete an adaptive card",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		if err := client.DeleteAdaptiveCard(ctx, args[0]); err != nil {
			return fmt.Errorf("delete card failed: %w", err)
		}
		if jsonOutput {
			printJSON(map[string]string{"status": "deleted", "cardId": args[0]})
		} else {
			fmt.Printf("Card %s deleted\n", args[0])
		}
		return nil
	},
}
