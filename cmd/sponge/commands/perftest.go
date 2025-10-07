package commands

import (
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/http"
	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/websocket"
)

// PerftestCommand command entry
func PerftestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perftest",
		Short: "Performance testing for HTTP/1.1, HTTP/2, HTTP/3, and websocket",
		Long: `Perftest is a performance testing tool that supports HTTP/1.1, HTTP/2, HTTP/3, and WebSocket protocols. It also allows real-time statistics
to be pushed to a custom HTTP endpoints or Prometheus, and supports two testing modes: standalone and distributed cluster testing.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		http.PerfTestHTTPCMD(),
		http.PerfTestHTTP2CMD(),
		http.PerfTestHTTP3CMD(),
		websocket.PerfTestWebsocketCMD(),

		http.PerfTestCollectorCMD(),
		http.PerfTestAgentCMD(),
	)

	return cmd
}
