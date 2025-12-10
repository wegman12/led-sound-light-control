package server

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/wegman12/led-sound-light-control/utilities"
	"go.uber.org/zap"
)

type serverConfig struct {
	host            string
	port            int
	readTimeout     int
	writeTimeout    int
	shutdownTimeout int
	debug           bool
}

var serverCfg serverConfig

func MakeServerCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the web server",
		Long:  `Starts the HTTP web server for LED sound light control`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			// Initialize logger
			logger, err := utilities.NewLogger(serverCfg.debug)
			if err != nil {
				return err
			}
			defer logger.Sync()

			logger.Info("Starting server",
				zap.String("host", serverCfg.host),
				zap.Int("port", serverCfg.port),
				zap.Bool("debug", serverCfg.debug),
			)

			config := ServerConfig{
				Host:            serverCfg.host,
				Port:            serverCfg.port,
				ReadTimeout:     serverCfg.readTimeout,
				WriteTimeout:    serverCfg.writeTimeout,
				ShutdownTimeout: serverCfg.shutdownTimeout,
			}

			server, err := NewServer(config, ctx, logger)
			if err != nil {
				logger.Fatal("Failed to create server", zap.Error(err))
				return err
			}
			return server.Start(ctx)
		},
	}

	cmd.Flags().StringVarP(&serverCfg.host, "host", "H", "0.0.0.0", "server host address")
	cmd.Flags().IntVarP(&serverCfg.port, "port", "p", 8080, "server port")
	cmd.Flags().IntVar(&serverCfg.readTimeout, "read-timeout", 10, "read timeout in seconds")
	cmd.Flags().IntVar(&serverCfg.writeTimeout, "write-timeout", 10, "write timeout in seconds")
	cmd.Flags().IntVar(&serverCfg.shutdownTimeout, "shutdown-timeout", 30, "graceful shutdown timeout in seconds")
	cmd.Flags().BoolVarP(&serverCfg.debug, "debug", "d", false, "enable debug logging")

	return cmd
}
