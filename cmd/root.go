package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shellchat",
	Short: "Zero-server P2P encrypted chat",
	Long:  `ShellChat is a peer-to-peer encrypted chat application with zero central server.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
