package light

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/wegman12/led-sound-light-control/utilities"
)

type ledTesterConfig struct {
	inputPath string
}

var ledCfg ledTesterConfig

func MakeLedTesterCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "test-led",
		Short: "Test led functionality from BBB board",
		Long:  `Runs led colors in a test loop`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
		RunE: func(cmd *cobra.Command, args []string) error {

			if !utilities.FileExists(ledCfg.inputPath) {
				return fmt.Errorf("ledTester input file does not exist: " + ledCfg.inputPath)
			}

			// Open the JSON file
			file, err := os.Open(ledCfg.inputPath)
			if err != nil {
				log.Fatalf("Error opening file: %v", err)
			}
			defer file.Close() // Ensure the file is closed

			var managerConfig ManagerConfig
			encoder := json.NewDecoder(file)
			err = encoder.Decode(&managerConfig)
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer stop() // Ensure the signal handling is stopped on exit

			// TODO: Phase 3 - Pass actual AudioProvider instead of nil
			mgr, err := NewManager(managerConfig, nil)
			if mgr != nil {
				defer mgr.Close()
			}
			if err != nil {
				return err
			}

			mgr.Start(ctx)
			<-ctx.Done()
			return nil
		},
	}

	cmd.Flags().StringVarP(&ledCfg.inputPath, "input", "i", "", "input file path")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
