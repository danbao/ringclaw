package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	userCmd.AddCommand(userSearchCmd)
	userCmd.AddCommand(userGetCmd)
	rootCmd.AddCommand(userCmd)
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User operations",
}

var userSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search company directory by name or email",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		query := strings.Join(args, " ")
		result, err := client.SearchDirectory(ctx, query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		if jsonOutput {
			printJSON(result)
		} else {
			fmt.Printf("Results (%d)\n", len(result.Records))
			for _, r := range result.Records {
				fmt.Printf("  %s  %s %s  %s\n", r.ID, r.FirstName, r.LastName, r.Email)
			}
		}
		return nil
	},
}

var userGetCmd = &cobra.Command{
	Use:   "get <personId>",
	Short: "Get person info",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newCLIClient()
		if err != nil {
			return err
		}
		ctx, cancel := notifyContext(context.Background())
		defer cancel()

		person, err := client.GetPersonInfo(ctx, args[0])
		if err != nil {
			return fmt.Errorf("get person failed: %w", err)
		}
		if jsonOutput {
			printJSON(person)
		} else {
			fmt.Printf("Person: %s\n", person.ID)
			fmt.Printf("  Name:  %s %s\n", person.FirstName, person.LastName)
			fmt.Printf("  Email: %s\n", person.Email)
		}
		return nil
	},
}
