package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "audio-util",
	Short: "PRU1 Audio Sampling Utility",
	Long: `A utility for testing and monitoring the PRU1 audio sampling firmware.

This tool provides commands to:
  - Test PRU1 basic operation
  - Monitor ADC sampling
  - View sound profile streams
  - Configure frequency bins

Use 'audio-util [command] --help' for more information about a command.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
