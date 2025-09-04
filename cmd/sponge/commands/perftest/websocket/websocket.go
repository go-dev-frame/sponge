package websocket

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
	"github.com/go-dev-frame/sponge/pkg/krand"
)

// PerfTestWebsocketCMD creates a cobra command for websocket performance test
func PerfTestWebsocketCMD() *cobra.Command {
	var (
		targetURL    string
		worker       int
		duration     time.Duration
		sendInterval time.Duration
		rampUp       time.Duration

		bodyString string
		bodyJSON   string
		bodyFile   string

		out string
	)

	cmd := &cobra.Command{
		Use:   "websocket",
		Short: "Run a performance test against WebSocket service",
		Long:  "Run a performance test against WebSocket service.",
		Example: color.HiBlackString(`  # Default: 10 workers, 10s duration, random(10) string message
  %s websocket --url=ws://localhost:8080/ws

  # Send fixed string messages, 100 workers, 1m duration, each worker sends messages every 10ms
  %s websocket --worker=100 --duration=1m --send-interval=10ms --body-string=abcdefghijklmnopqrstuvwxyz --url=ws://localhost:8080/ws

  # Send JSON messages, 10 workers, 10s duration
  %s websocket --worker=10 --duration=10s --body={\"name\":\"Alice\",\"age\":25} --url=ws://localhost:8080/ws

  # Send JSON messages, 100 workers, 1m duration, each worker sends messages every 10ms
  %s websocket --worker=100 --duration=1m --send-interval=10ms --body={\"name\":\"Alice\",\"age\":25} --url=ws://localhost:8080/ws`,
			common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := common.CheckBodyParam(bodyJSON, bodyFile)
			if err != nil {
				return err
			}

			var payloadData []byte
			isJSON := false
			if body != "" {
				payloadData = []byte(body)
				isJSON = true
			} else {
				if bodyString != "" {
					payloadData = []byte(bodyString)
				} else {
					payloadData = krand.Bytes(krand.R_All, 10) // default payload data random(10)
				}
			}

			p := &perfTestParams{
				targetURL:    targetURL,
				worker:       worker,
				duration:     duration,
				sendInterval: sendInterval,
				rampUp:       rampUp,
				payloadData:  payloadData,
				isJSON:       isJSON,
				out:          out,
			}

			return p.run()
		},
	}

	cmd.Flags().StringVarP(&targetURL, "url", "u", "", "request URL")
	_ = cmd.MarkFlagRequired("url")
	cmd.Flags().IntVarP(&worker, "worker", "c", 10, "number of concurrent websocket clients")
	cmd.Flags().DurationVarP(&duration, "duration", "d", time.Second*10, "duration of the test, e.g., 10s, 1m")
	cmd.Flags().DurationVarP(&sendInterval, "send-interval", "i", 0, "interval for sending messages per client")
	cmd.Flags().DurationVarP(&rampUp, "ramp-up", "r", 0, "time to ramp up all connections (e.g., 10s)")

	cmd.Flags().StringVarP(&bodyJSON, "body", "b", "", "request body (JSON String, priority higher than --body-file, --body-string)")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "f", "", "request body file")
	cmd.Flags().StringVarP(&bodyString, "body-string", "s", "", "request body (String)")

	cmd.Flags().StringVarP(&out, "out", "o", "", "save statistics to JSON file")

	return cmd
}

type perfTestParams struct {
	targetURL    string
	worker       int
	duration     time.Duration
	sendInterval time.Duration
	rampUp       time.Duration

	payloadData []byte
	isJSON      bool

	out string
}

func (p *perfTestParams) run() error {
	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test duration timer
	go func() {
		<-time.After(p.duration)
		cancel()
	}()

	// OS interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-interrupt
		cancel()
	}()

	// Calculate ramp-up delay
	var rampUpDelay time.Duration
	if p.worker > 0 && p.rampUp > 0 {
		rampUpDelay = p.rampUp / time.Duration(p.worker)
	}

	stats := &statsCollector{errSet: NewErrSet()}
	bar := common.NewTimeBar(p.duration)
	bar.Start()

	// Start all client workers with ramp-up
	var wg sync.WaitGroup
	for i := 0; i < p.worker; i++ {
		if mainCtx.Err() != nil {
			break
		}

		client := NewClient(i+1, p.targetURL, stats, p.sendInterval, p.payloadData, p.isJSON)
		wg.Go(func() { client.Run(mainCtx) })

		if rampUpDelay > 0 {
			time.Sleep(rampUpDelay)
		}
	}

	// Wait for all workers to finish
	wg.Wait()
	if !errors.Is(mainCtx.Err(), context.Canceled) {
		bar.Finish()
	}
	fmt.Println()

	st := stats.PrintReport(p.duration, p.targetURL)
	if p.out != "" && st != nil {
		err := st.Save(p.out)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("failed to save statistics to file: %v", err)
		}
		fmt.Printf("\nsave statistics to '%s' successfully\n", p.out)
	}

	return nil
}
