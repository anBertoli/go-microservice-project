package main

import (
	"github.com/spf13/cobra"
	"log"
)

// Define the root command of the CLI.
var rootCmd = &cobra.Command{
	Use:   "snap-cli",
	Short: "Snap Vault CLI",
	Long:  `Snap Vault CLI to perform system and admin operations`,
}

func main() {
	// Register the migrate command.
	initMigrateCmd()

	// Start parsing the command line arguments and execute the appropriate command.
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
