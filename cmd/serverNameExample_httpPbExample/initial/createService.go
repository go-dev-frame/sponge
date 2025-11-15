package initial

import (
	"strconv"

	"github.com/go-dev-frame/sponge/internal/config"
	"github.com/go-dev-frame/sponge/internal/server"

	"github.com/go-dev-frame/sponge/pkg/app"
)

// CreateServices create http service
func CreateServices() []app.IServer {
	var cfg = config.Get()
	var servers []app.IServer

	// create a http service
	httpAddr := ":" + strconv.Itoa(cfg.HTTP.Port)
	httpServer := server.NewHTTPServer_pbExample(httpAddr,
		server.WithHTTPIsProd(cfg.App.Env == "prod"),
		server.WithHTTPTLS(cfg.HTTP.TLS),
	)
	servers = append(servers, httpServer)

	return servers
}
