package cmd

import (
	"fmt"
	"syscall"

	"shellchat/storage"

	"golang.org/x/term"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/argon2"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the encrypted database",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Enter new master password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("\nError reading password:", err)
			return
		}
		password := string(bytePassword)
		fmt.Println()

		fmt.Print("Confirm master password: ")
		byteConfirm, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("\nError reading password:", err)
			return
		}
		confirm := string(byteConfirm)
		fmt.Println()

		if password != confirm {
			fmt.Println("Passwords do not match.")
			return
		}

		// Derive key using Argon2 (for demonstration, we use the password directly as key for SQLCipher pragma,
		// but typically you'd salt and hash it. SQLCipher handles key derivation internally if provided a raw key,
		// but providing a consistent hashed key is also good practice if we want to change KDF settings manually.
		// However, SQLCipher's default KDF is PBKDF2.
		// The prompt requirement says: "Hash this password using argon2 to derive the AES encryption key".
		// SQLCipher `PRAGMA key` accepts a hex string for raw key.
		// Let's derive a 32-byte key and encode as known format for SQLCipher.

		salt := []byte("shellchat-static-salt") // In production, this should be random and stored.
		key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
		hexKey := fmt.Sprintf("x'%x'", key)

		// Initialize DB with the derived key
		if err := storage.InitDB(hexKey); err != nil {
			fmt.Println("Failed to initialize database:", err)
			return
		}
		defer storage.CloseDB()

		fmt.Println("Database initialized and encrypted successfully.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
