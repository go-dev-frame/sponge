package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// ---------------------------------- client interceptor ----------------------------------

var timeoutVal = time.Second * 10

// UnaryClientTimeout client-side timeout unary interceptor
func UnaryClientTimeout(d time.Duration) grpc.UnaryClientInterceptor {
	if d < time.Millisecond {
		d = timeoutVal
	}

	return func(ctx context.Context, method string, req, resp interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

// StreamClientTimeout server-side timeout  interceptor
func StreamClientTimeout(d time.Duration) grpc.StreamClientInterceptor {
	if d < time.Millisecond {
		d = timeoutVal
	}

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return streamer(ctx, desc, cc, method, opts...)
	}
}
