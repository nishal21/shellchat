package cmd

import (
	"fmt"
	"os"
	"shellchat/p2p"
	"shellchat/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start the chat UI",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize P2P Host
		// Use random port 0 to let OS choose, or fixed for dev?
		// Peer discovery needs to know port? mDNS handles it.
		// Randomness for key generation

		fmt.Println("Initializing P2P Node...")
		h, err := p2p.MakeHost(0, nil) // nil randomness = crypto/rand
		if err != nil {
			fmt.Printf("Failed to create host: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("I am %s\n", h.P2PHost.ID().String())

		// Start Discovery
		if err := p2p.SetupDiscovery(h.P2PHost, h.DHT); err != nil {
			fmt.Printf("Failed to start discovery: %v\n", err)
		}

		p := tea.NewProgram(ui.InitialModel(h))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
}
