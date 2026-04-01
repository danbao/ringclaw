package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	chatListCmd.Flags().StringVar(&chatTypeFilter, "type", "", "Filter by chat type (Direct, Team, Group, Personal, Everyone)")
	chatListCmd.Flags().BoolVar(&chatRecent, "recent", false, "Sort by most recently active")

	chatCmd.AddCommand(chatListCmd)
	chatCmd.AddCommand(chatGetCmd)
	rootCmd.AddCommand(chatCmd)
}

var (
	chatTypeFilter string
	chatRecent     bool
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat operations",
}

var chatListCmd = &cobra.Command{
	Use:   "list",
	Short: "List chats",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		if chatRecent {
			list, err := client.ListRecentChats(ctx, chatTypeFilter, 50)
			if err != nil {
				return fmt.Errorf("list recent chats failed: %w", err)
			}
			if jsonOutput {
				printJSON(list)
			} else {
				fmt.Printf("Recent Chats (%d)\n", len(list.Records))
				for _, c := range list.Records {
					fmt.Printf("  %s  %-8s  %s\n", c.ID, c.Type, c.Name)
				}
			}
			return nil
		}

		list, err := client.ListChats(ctx, chatTypeFilter)
		if err != nil {
			return fmt.Errorf("list chats failed: %w", err)
		}
		if jsonOutput {
			printJSON(list)
		} else {
			fmt.Printf("Chats (%d)\n", len(list.Records))
			for _, c := range list.Records {
				fmt.Printf("  %s  %-8s  %s\n", c.ID, c.Type, c.Name)
			}
		}
		return nil
	},
}

var chatGetCmd = &cobra.Command{
	Use:   "get <chatId>",
	Short: "Get chat details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		chat, err := client.GetChat(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get chat failed: %w", err)
		}
		if jsonOutput {
			printJSON(chat)
		} else {
			fmt.Printf("Chat: %s\n", chat.ID)
			fmt.Printf("  Name:    %s\n", chat.Name)
			fmt.Printf("  Type:    %s\n", chat.Type)
			if chat.Description != "" {
				fmt.Printf("  Desc:    %s\n", chat.Description)
			}
			fmt.Printf("  Members: %d\n", len(chat.Members))
			if chat.Status != "" {
				fmt.Printf("  Status:  %s\n", chat.Status)
			}
			fmt.Printf("  Created: %s\n", chat.CreationTime)
		}
		return nil
	},
}
