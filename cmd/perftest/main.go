package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/grpc"
	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/http"
	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/websocket"
)

func main() {
	cmd := perftestCommand()
	if err := cmd.Execute(); err != nil {
		cmd.PrintErrln("Error:", err)
		os.Exit(1)
	}
}

func perftestCommand() *cobra.Command {
	common.SetCommandPrefix("perftest")
	cmd := &cobra.Command{
		Use:   "perftest",
		Short: "Performance testing for HTTP/1.1, HTTP/2, HTTP/3, WebSocket and gRPC",
		Long: `Perftest is a high-performance testing tool that supports HTTP/1.1, HTTP/2, HTTP/3, WebSocket and gRPC protocols.  
It provides real-time metrics reporting to custom HTTP endpoints or Prometheus, and supports two modes:  
standalone testing and distributed cluster testing.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		http.PerfTestHTTPCMD(),
		http.PerfTestHTTP2CMD(),
		http.PerfTestHTTP3CMD(),
		websocket.PerfTestWebsocketCMD(),
		grpc.PerfTestGRPCCMD(),

		http.PerfTestCollectorCMD(),
		http.PerfTestAgentCMD(),
	)

	return cmd
}
