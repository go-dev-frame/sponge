package service

import (
	"context"
	"io"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/go-dev-frame/sponge/pkg/grpc/grpccli"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"github.com/go-dev-frame/sponge/configs"
	"github.com/go-dev-frame/sponge/internal/config"
)

var ioEOF = io.EOF

func TestRegisterAllService(t *testing.T) {
	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		server := grpc.NewServer()
		RegisterAllService(server)
		cancel()
	})
}

// The default is to connect to the local grpc server, if you want to connect to a remote grpc server,
// pass in the parameter grpcClient.
func getRPCClientConnForTest(grpcClient ...config.GrpcClient) *grpc.ClientConn {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		panic(err)
	}
	grpcClientCfg := getGRPCClientCfg(grpcClient...)

	var cliOptions []grpccli.Option

	endpoint := grpcClientCfg.Host + ":" + strconv.Itoa(grpcClientCfg.Port)
	isUseDiscover := false

	// using service discovery
	//discoverOption, discoveryEndpoint := discoverService(config.Get(), grpcClientCfg)
	//if discoverOption != nil {
	//	isUseDiscover = true
	//	endpoint = discoveryEndpoint
	//	cliOptions = append(cliOptions, discoverOption)
	//	cliOptions = append(cliOptions, grpccli.WithEnableLoadBalance()) // load balance
	//}

	if grpcClientCfg.Timeout > 0 {
		cliOptions = append(cliOptions, grpccli.WithTimeout(time.Second*time.Duration(grpcClientCfg.Timeout)))
	}

	// secure
	cliOptions = append(cliOptions, grpccli.WithSecure(
		grpcClientCfg.ClientSecure.Type,
		grpcClientCfg.ClientSecure.ServerName,
		grpcClientCfg.ClientSecure.CaFile,
		grpcClientCfg.ClientSecure.CertFile,
		grpcClientCfg.ClientSecure.KeyFile,
	))

	// token
	cliOptions = append(cliOptions, grpccli.WithToken(
		grpcClientCfg.ClientToken.Enable,
		grpcClientCfg.ClientToken.AppID,
		grpcClientCfg.ClientToken.AppKey,
	))

	cliOptions = append(cliOptions,
		grpccli.WithEnableRequestID(),
		grpccli.WithEnableLog(logger.Get()),
	)

	msg := "dialing grpc server"
	if isUseDiscover {
		msg += " with discovery from " + grpcClientCfg.RegistryDiscoveryType
	}
	logger.Info(msg, logger.String("name", grpcClientCfg.Name), logger.String("endpoint", endpoint))

	conn, err := grpccli.NewClient(endpoint, cliOptions...)
	if err != nil {
		panic(err)
	}

	return conn
}

func getGRPCClientCfg(grpcClient ...config.GrpcClient) config.GrpcClient {
	var grpcClientCfg config.GrpcClient

	// custom config
	if len(grpcClient) > 0 {
		// parameter config, highest priority
		grpcClientCfg = grpcClient[0]
	} else {
		// grpcClient config in the yaml file, second priority
		if len(config.Get().GrpcClient) > 0 {
			for _, v := range config.Get().GrpcClient {
				if v.Name == config.Get().App.Name { // match the current app name
					grpcClientCfg = v
					break
				}
			}
		}
	}

	// if there is no custom configuration, use the default configuration
	if grpcClientCfg.Name == "" {
		grpcClientCfg = config.GrpcClient{
			Host: config.Get().App.Host,
			Port: config.Get().Grpc.Port,
			// If RegistryDiscoveryType is not empty, service discovery is used, and Host and Port values are invalid
			RegistryDiscoveryType: config.Get().App.RegistryDiscoveryType, // supports consul, etcd and nacos
			Name:                  config.Get().App.Name,
		}
	}

	return grpcClientCfg
}

// discovery service with consul or etcd or nacos, select one of them to use
//func discoverService(cfg *config.Config, grpcClientCfg config.GrpcClient) (grpccli.Option, string) {
//	var (
//		endpoint      string
//		grpcCliOption grpccli.Option
//	)
//
//	switch grpcClientCfg.RegistryDiscoveryType {
//	case "consul":
//		endpoint = "discovery:///" + grpcClientCfg.Name // format: discovery:///serverName.scheme
//		cli, err := consulcli.Init(cfg.Consul.Addr, consulcli.WithWaitTime(time.Second*2))
//		if err != nil {
//			panic(err)
//		}
//		iDiscovery := consul.New(cli)
//		grpcCliOption = grpccli.WithDiscovery(iDiscovery)
//
//	case "etcd":
//		endpoint = "discovery:///" + grpcClientCfg.Name // format: discovery:///serverName.scheme
//		cli, err := etcdcli.Init(cfg.Etcd.Addrs, etcdcli.WithDialTimeout(time.Second*2))
//		if err != nil {
//			panic(err)
//		}
//		iDiscovery := etcd.New(cli)
//		grpcCliOption = grpccli.WithDiscovery(iDiscovery)
//
//	case "nacos":
//		endpoint = "discovery:///" + grpcClientCfg.Name + ".grpc" // format: discovery:///serverName.scheme
//		cli, err := nacoscli.NewNamingClient(
//			cfg.NacosRd.IPAddr,
//			cfg.NacosRd.Port,
//			cfg.NacosRd.NamespaceID)
//		if err != nil {
//			panic(err)
//		}
//		iDiscovery := nacos.New(cli)
//		grpcCliOption = grpccli.WithDiscovery(iDiscovery)
//	}
//
//	return grpcCliOption, endpoint
//}
