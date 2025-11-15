package initial

import (
	"strconv"

	"github.com/go-dev-frame/sponge/pkg/app"

	"github.com/go-dev-frame/sponge/internal/config"
	"github.com/go-dev-frame/sponge/internal/server"
)

// CreateServices create http service
func CreateServices() []app.IServer {
	var cfg = config.Get()
	var servers []app.IServer
	var httpAddr = ":" + strconv.Itoa(cfg.HTTP.Port)

	// case 1, create a http service without registry
	httpServer := server.NewHTTPServer_pbExample(httpAddr,
		server.WithHTTPIsProd(cfg.App.Env == "prod"),
		server.WithHTTPTLS(cfg.HTTP.TLS),
	)

	// case 2, Create a http service and register it with consul or etcd or nacos
	//httpRegistry, httpInstance := registerService("http", cfg.App.Host, cfg.HTTP.Port)
	//httpServer := server.NewHTTPServer_pbExample(httpAddr,
	//	server.WithHTTPRegistry(httpRegistry, httpInstance),
	//	server.WithHTTPIsProd(cfg.App.Env == "prod"),
	//	server.WithHTTPTLS(cfg.HTTP.TLS),
	//)

	servers = append(servers, httpServer)

	return servers
}

// register service with consul or etcd or nacos, select one of them to use
//func registerService(scheme string, host string, port int) (registry.Registry, *registry.ServiceInstance) {
//	var (
//		instanceEndpoint = fmt.Sprintf("%s://%s:%d", scheme, host, port)
//		cfg              = config.Get()
//
//		iRegistry registry.Registry
//		instance  *registry.ServiceInstance
//		err       error
//
//		id       = cfg.App.Name + "_" + scheme + "_" + host + "_" + strconv.Itoa(port)
//		logField logger.Field
//	)
//
//	switch cfg.App.RegistryDiscoveryType {
//	case "consul":
//		iRegistry, instance, err = consul.NewRegistry(
//			cfg.Consul.Addr,
//			id,
//			cfg.App.Name,
//			[]string{instanceEndpoint},
//		)
//		if err != nil {
//			panic(err)
//		}
//		logField = logger.Any("consulAddress", cfg.Consul.Addr)
//
//	case "etcd":
//		iRegistry, instance, err = etcd.NewRegistry(
//			cfg.Etcd.Addrs,
//			id,
//			cfg.App.Name,
//			[]string{instanceEndpoint},
//		)
//		if err != nil {
//			panic(err)
//		}
//		logField = logger.Any("etcdAddress", cfg.Etcd.Addrs)
//
//	case "nacos":
//		iRegistry, instance, err = nacos.NewRegistry(
//			cfg.NacosRd.IPAddr,
//			cfg.NacosRd.Port,
//			cfg.NacosRd.NamespaceID,
//			id,
//			cfg.App.Name,
//			[]string{instanceEndpoint},
//		)
//		if err != nil {
//			panic(err)
//		}
//		logField = logger.String("nacosAddress", fmt.Sprintf("%v:%d", cfg.NacosRd.IPAddr, cfg.NacosRd.Port))
//	}
//
//	if instance != nil {
//		msg := fmt.Sprintf("register service address to %s", cfg.App.RegistryDiscoveryType)
//		logger.Info(msg, logger.String("name", cfg.App.Name), logger.String("endpoint", instanceEndpoint), logger.String("id", id), logField)
//		return iRegistry, instance
//	}
//
//	return nil, nil
//}
