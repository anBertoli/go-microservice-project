package main

import (
	"github.com/spf13/cobra"
	"log"
)

var rootCmd = &cobra.Command{
	Use:   "snap-cli",
	Short: "Snap Vault CLI",
	Long:  `Snap Vault CLI to perform system and admin operations`,
}

func main() {
	initMigrateCmd()

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
