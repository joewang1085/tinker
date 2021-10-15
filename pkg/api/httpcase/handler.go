package httpcase

import (
	"context"

	"tinker/mock/pb/hello"
	"tinker/pkg/framework"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type httpCase struct {
	grpcAddrs []string
}

func NewHttpCase(grpcAddr ...string) *httpCase {
	return &httpCase{
		grpcAddrs: grpcAddr,
	}
}

func (p *httpCase) Handler() *framework.Handler {
	ret := framework.DefaultHttpHandler("httpCase", p.grpcAddrs)

	ret.Add(p.CallGRPC)

	return ret
}

// CallGRPC creates a client
func (p *httpCase) CallGRPC(sess *framework.Session) error {
	// 此处仅模拟使用1个grpc conn
	c := hello.NewGreetingClient(sess.GrpcConns[0])

	// 发起请求
	request := &hello.GreetRequest{
		Saying: "hello",
		Person: hello.Name_Joe,
		Time:   timestamppb.Now(),
		Fruit:  hello.GreetRequest_apple,
	}

	response, err := c.Greet(context.Background(), request)
	if err != nil {
		sess.Errorf("CallGRPC: fail to call grpc: %s", err.Error())
		return err
	}

	err = framework.SendHttpResult(sess, response)
	if err != nil {
		sess.Errorf("CallGRPC: fail to send response to client: %s", err.Error())
		return err
	}

	return nil
}
