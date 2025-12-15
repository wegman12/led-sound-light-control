package audio

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/wegman12/led-sound-light-control/audio/processing"
	"github.com/wegman12/led-sound-light-control/utilities"
	"go.uber.org/zap"
)

type soundTesterConfig struct {
	bufferSize             int
	samplingRate           int
	targetInputRate        float64
	baseCutoff             float64
	midCutoff              float64
	delayBetweenSamples    time.Duration
	delayBetweenProcessing time.Duration
	outputFile             string
	debug                  bool
}

var cfg soundTesterConfig

func MakeSoundTesterCmd() *cobra.Command {
	soundTester := &cobra.Command{
		Use:   "test-sound",
		Short: "Test sound functionality from BBB board",
		Long:  `Runs sound sampling in a loop and displays the fft results`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
		RunE: doSoundTester,
	}

	soundTester.Flags().IntVarP(&cfg.bufferSize, "buffer-size", "p", 1024, "buffer size in bytes")
	soundTester.Flags().IntVarP(&cfg.samplingRate, "sampling-rate", "s", 48000, "sampling rate")
	soundTester.Flags().Float64Var(&cfg.targetInputRate, "target-input-rate", 200000.0, "expected un-decimated sampling rate")
	soundTester.Flags().Float64Var(&cfg.baseCutoff, "base-cutoff", 120.0, "base cutoff")
	soundTester.Flags().Float64Var(&cfg.midCutoff, "mid-cutoff", 2000.0, "mid cutoff")
	soundTester.Flags().DurationVar(&cfg.delayBetweenProcessing, "delay-between-processing", time.Millisecond*1, "delay between processing processing")
	soundTester.Flags().DurationVar(&cfg.delayBetweenSamples, "delay-between-samples", time.Millisecond*1, "delay between samples processing")
	soundTester.Flags().StringVarP(&cfg.outputFile, "output-file", "o", "sound-test-result.csv", "output file to write the results to")
	soundTester.Flags().BoolVarP(&cfg.debug, "debug", "d", false, "debug mode")

	return soundTester
}

func doSoundTester(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	var logger *zap.Logger
	var err error
	if cfg.debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Println("failed to initialize logger - using nop logger")
		fmt.Println(err)
		logger = zap.NewNop()
	}

	m, err := NewManager(
		cfg.bufferSize,
		cfg.samplingRate,
		cfg.targetInputRate,
		cfg.baseCutoff,
		cfg.midCutoff,
		cfg.delayBetweenSamples,
		cfg.delayBetweenProcessing,
		logger,
	)

	if err != nil {
		logger.Fatal("failed to create sound manager", zap.Error(err))
	}

	wg := &sync.WaitGroup{}

	resultChannel := make(chan processing.Result, 10000)

	wg.Add(2)
	go func() {
		logger.Info("starting processing manager")
		defer wg.Done()
		m.Start(resultChannel, ctx)
		logger.Info("finished processing manager")
	}()

	go func() {
		logger.Info("starting export manager")
		defer wg.Done()
		exportResults(ctx, resultChannel, logger)
		logger.Info("export finished")
	}()

	wg.Wait()
	return nil
}

func exportResults(ctx context.Context, resultChannel chan processing.Result, logger *zap.Logger) {
	f, err := os.Create(cfg.outputFile)
	if err != nil {
		logger.Error("failed to create output file", zap.Error(err))
		return
	}
	writeHeader(f)
	defer func() {
		err := f.Close()
		logger.Info("results written", zap.String("file", cfg.outputFile))
		if err != nil {
			logger.Error("failed to close file", zap.Error(err))
		}
	}()
	totalDuration := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("export finished")
			return
		case result := <-resultChannel:
			stringResults := []string{
				fmt.Sprintf("%v", totalDuration.Milliseconds()),
				fmt.Sprintf("%v", result.SignalStrength),
				fmt.Sprintf("%v", result.Profile.Bass),
				fmt.Sprintf("%v", result.Profile.Mid),
				fmt.Sprintf("%v", result.Profile.Treble),
			}
			totalDuration += result.SamplingDuration
			magnitudes := utilities.Apply(result.Magnitudes[:], func(v float64) string { return fmt.Sprint(v) })
			stringResults = append(stringResults, magnitudes...)
			_, err = f.WriteString(strings.Join(stringResults, ",") + "\n")
			if err != nil {
				logger.Error("failed to write results to file", zap.Error(err))
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type pruRecordConfig struct {
	outputFile     string
	sampleInterval time.Duration
	duration       time.Duration
	debug          bool
}

var pruRecordCfg pruRecordConfig

func MakePRURecordCmd() *cobra.Command {
	pruRecord := &cobra.Command{
		Use:   "record-pru-audio",
		Short: "Record PRU audio stream to CSV file",
		Long:  `Continuously reads from the PRU1 audio sampler and writes sound profile data to a CSV file for analysis`,
		RunE:  doPRURecord,
	}

	pruRecord.Flags().StringVarP(&pruRecordCfg.outputFile, "output", "o", "pru-audio-record.csv", "output CSV file path")
	pruRecord.Flags().DurationVarP(&pruRecordCfg.sampleInterval, "interval", "i", 25*time.Millisecond, "sampling interval (time between reads)")
	pruRecord.Flags().DurationVarP(&pruRecordCfg.duration, "duration", "t", 0, "recording duration (0 for unlimited, runs until Ctrl+C)")
	pruRecord.Flags().BoolVarP(&pruRecordCfg.debug, "debug", "d", false, "enable debug logging")

	return pruRecord
}

func doPRURecord(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	// Add timeout if duration is specified
	if pruRecordCfg.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, pruRecordCfg.duration)
		defer cancel()
	}

	// Initialize logger
	var logger *zap.Logger
	var err error
	if pruRecordCfg.debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Println("failed to initialize logger - using nop logger")
		fmt.Println(err)
		logger = zap.NewNop()
	}
	defer logger.Sync()

	// Create PRU manager
	manager, err := NewPRUManager(logger)
	if err != nil {
		logger.Fatal("failed to create PRU manager", zap.Error(err))
	}
	defer manager.Close()

	// Create output file
	f, err := os.Create(pruRecordCfg.outputFile)
	if err != nil {
		logger.Fatal("failed to create output file", zap.Error(err))
	}
	defer f.Close()

	// Write CSV header
	writePRUHeader(f)

	logger.Info("Starting PRU audio recording",
		zap.String("output_file", pruRecordCfg.outputFile),
		zap.Duration("sample_interval", pruRecordCfg.sampleInterval),
		zap.Duration("duration", pruRecordCfg.duration),
	)

	// Create channel for sound profiles
	profileChannel := make(chan *SoundProfile, 100)

	// Start the PRU manager
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := manager.Start(profileChannel, pruRecordCfg.sampleInterval, ctx)
		if err != nil {
			logger.Error("PRU manager error", zap.Error(err))
		}
	}()

	// Record data to CSV
	recordCount := 0
	startTime := time.Now()

loop:
	for {
		select {
		case <-ctx.Done():
			logger.Info("Recording stopped", zap.Int("samples_recorded", recordCount))
			break loop

		case profile := <-profileChannel:
			elapsed := time.Since(startTime)

			// Write CSV row
			row := fmt.Sprintf("%d,%f,%d,%.2f,%.3f,%d,%d,%d,%d,%d,%d,%d,%d\n",
				recordCount,
				elapsed.Seconds(),
				profile.FFTCount,
				profile.FFTRate,
				profile.FFTTimeMs,
				profile.BassSum,
				profile.MidLowSum,
				profile.MidHighSum,
				profile.TrebleSum,
				profile.BassAvg,
				profile.MidLowAvg,
				profile.MidHighAvg,
				profile.TrebleAvg,
			)

			if _, err := f.WriteString(row); err != nil {
				logger.Error("failed to write to CSV", zap.Error(err))
			}

			recordCount++
			if recordCount%100 == 0 {
				logger.Info("Recording progress", zap.Int("samples", recordCount))
			}
		}
	}

	wg.Wait()
	logger.Info("Recording complete",
		zap.Int("total_samples", recordCount),
		zap.Duration("total_duration", time.Since(startTime)),
		zap.String("output_file", pruRecordCfg.outputFile),
	)

	return nil
}

func writePRUHeader(f *os.File) {
	headers := []string{
		"Sample",
		"Timestamp (s)",
		"FFT Count",
		"FFT Rate (Hz)",
		"FFT Time (ms)",
		"Bass Sum",
		"Mid-Low Sum",
		"Mid-High Sum",
		"Treble Sum",
		"Bass Avg",
		"Mid-Low Avg",
		"Mid-High Avg",
		"Treble Avg",
	}

	_, err := f.WriteString(strings.Join(headers, ",") + "\n")
	if err != nil {
		fmt.Println(err)
	}
}

func writeHeader(f *os.File) {
	binSize := float64(cfg.samplingRate) / float64(cfg.bufferSize)
	binDescriptions := make([]string, cfg.bufferSize/2-1)
	binLower := binSize
	for i := 1; i < cfg.bufferSize/2; i++ {
		binDescriptions[i-1] = fmt.Sprintf("%0.2f-%0.2f KHZ", binLower/1000, (binLower+binSize)/1000)
		binLower += binSize
	}

	headers := []string{
		"Timestamp (ms)",
		"Signal Strength",
		"Bass",
		"Mid",
		"Treble",
	}

	headers = append(headers, binDescriptions...)

	_, err := f.WriteString(strings.Join(headers, ",") + "\n")
	if err != nil {
		fmt.Println(err)
	}
}
