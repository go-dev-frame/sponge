## grpccli

`grpccli` is Go's gRPC client library that supports for service discovery, logging, load balancing, trace, metrics, retries, circuit breaker.

### Example of use

```go
    import "github.com/go-dev-frame/sponge/pkg/grpc/grpccli"

func grpcClientExample() serverNameV1.UserExampleServiceClient {
	err := config.Init(third_party.Path("../config/conf.yml"))
	if err != nil {
		panic(err)
	}

	var discovery registry.Discovery
	var endpoint = fmt.Sprintf("127.0.0.1:%d", config.Get().Grpc.Port)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if config.Get().App.EnableRegistryDiscovery {
		endpoint = "discovery:///" + config.Get().App.Name
		discovery = discoveryETCD(config.Get().Etcd.Addrs)
	}
	conn, err := grpccli.DialInsecure(ctx, endpoint,
		grpccli.WithEnableLog(logger.Get()),
		grpccli.WithDiscovery(discovery),
        //grpccli.WithEnableCircuitBreaker(),		
		//grpccli.WithEnableTrace(),
		//grpccli.WithEnableLoadBalance(),
		//grpccli.WithEnableRetry(),
		//grpccli.WithEnableMetrics(),
	)
	if err != nil {
		panic(err)
	}

	return serverNameV1.NewUserExampleServiceClient(conn)
}

func discoveryETCD(endpoints []string) registry.Discovery {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 10 * time.Second,
		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	})
	if err != nil {
		panic(err)
	}

	return etcd.New(cli)
}
```
