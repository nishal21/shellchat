package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"shellchat/storage"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

var clearHistoryCmd = &cobra.Command{
	Use:   "clearhistory",
	Short: "Clear all chat history from the local database",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Enter master password to authorize clearing history: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("\nError reading password:", err)
			return
		}
		password := string(bytePassword)
		fmt.Println()

		// Derive key (same logic as init)
		salt := []byte("shellchat-static-salt")
		key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
		hexKey := fmt.Sprintf("x'%x'", key)

		// Open DB
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Println("Error finding config directory:", err)
			return
		}

		if err := storage.InitDB(userConfigDir, hexKey); err != nil {
			fmt.Println("Failed to open database (wrong password?):", err)
			return
		}
		defer storage.CloseDB()

		if err := storage.ClearHistory(); err != nil {
			fmt.Println("Failed to clear history:", err)
			return
		}

		fmt.Println("Chat history cleared successfully.")
	},
}

var obliterateCmd = &cobra.Command{
	Use:   "obliterate",
	Short: "Delete the entire encrypted database file",
	Long:  `Safely deletes the entire encrypted database file from the disk and removes all trace of the application's data.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Are you sure you want to delete all data? This cannot be undone. (y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Operation cancelled.")
			return
		}

		configDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Println("Error finding config directory:", err)
			return
		}

		dbPath := filepath.Join(configDir, "shellchat", "shellchat.db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("Database file does not exist.")
			// Also check directory
			return
		}

		if err := os.Remove(dbPath); err != nil {
			fmt.Println("Error deleting database file:", err)
			return
		}

		fmt.Println("Database obliterated.")
	},
}

func init() {
	rootCmd.AddCommand(clearHistoryCmd)
	rootCmd.AddCommand(obliterateCmd)
}
