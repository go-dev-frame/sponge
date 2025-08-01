package initial

import (
	"context"
	"time"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/tracer"

	"github.com/go-dev-frame/sponge/internal/config"
	//"github.com/go-dev-frame/sponge/internal/rpcclient"
)

// Close releasing resources after service exit
func Close(servers []app.IServer) []app.Close {
	var closes []app.Close

	// close server
	for _, s := range servers {
		closes = append(closes, s.Stop)
	}

	// close the rpc client connection
	// example:
	//closes = append(closes, func() error {
	//	return rpcclient.CloseServerNameExampleRPCConn()
	//})

	// close tracing
	if config.Get().App.EnableTrace {
		closes = append(closes, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			return tracer.Close(ctx)
		})
	}

	// close logger
	closes = append(closes, func() error {
		return logger.Sync()
	})

	return closes
}
