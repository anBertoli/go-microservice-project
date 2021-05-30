package main

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

// Define a new migrate command in our CLI.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "convenient golang-migrate wrapper to migrate PG database",
	Run:   execMigrateCmd,
}

// Register the command to the main command of the CLI.
func initMigrateCmd() {
	flags := migrateCmd.Flags()
	flags.String("database-url", "postgres://localhost:5432/snapvault?sslmode=disable", "database url (ex: postgres://localhost:5432/database?sslmode=disable)")
	flags.String("migrations-folder", "file://migrations", "url path containing the migrations (ex: file://db/migrations-pg)")
	flags.String("action", "up", "possible value: 'up', 'down' or 'drop'")
	flags.IntP("version-to-force", "f", 0, "version value to be forced")
	rootCmd.AddCommand(migrateCmd)
}

// Execute the logic of the migrate command.
func execMigrateCmd(cmd *cobra.Command, args []string) {
	folder, err := cmd.Flags().GetString("migrations-folder")
	if err != nil {
		log.Fatal(err)
	}
	dbURL, err := cmd.Flags().GetString("database-url")
	if err != nil {
		log.Fatal(err)
	}
	action, err := cmd.Flags().GetString("action")
	if err != nil {
		log.Fatal(err)
	}
	version, err := cmd.Flags().GetInt("version-to-force")
	if err != nil {
		log.Fatal(err)
	}

	migrator, err := migrate.New(folder, dbURL)
	if err != nil {
		log.Fatalf("error executing the migration: %v", err)
	}
	defer func() {
		srcError, dbError := migrator.Close()
		if srcError != nil {
			log.Printf("error closing src: %v\n", srcError)
		}
		if dbError != nil {
			log.Printf("error closing database: %v\n", srcError)
		}
	}()
	migrator.Log = migrationLogger{log.Default()}

	switch action {
	case "up":
		err = migrator.Up()
	case "down":
		err = migrator.Down()
	case "drop":
		err = migrator.Down()
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("error during down phase (before drop): %v", err)
		}
		err = migrator.Drop()
	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			log.Fatalf("error applying migrations: %v", err)
		}
		fmt.Printf("version: %v, dirty: %v\n", version, dirty)
	case "force":
		err := migrator.Force(version)
		if err != nil {
			log.Fatalf("error applying migrations: %v", err)
		}
	}
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("error applying migrations: %v", err)
	}

	log.Print("done")
}

type migrationLogger struct {
	*log.Logger
}

func (ml migrationLogger) Verbose() bool {
	return false
}
