package websocket

import (
	"context"
	"crypto/rand"
	"fmt"

	"tinker/mock/pb/hello"
	"tinker/pkg/framework"
)

// 与 NewWebsocket 注册时保持一致
var gprcClientKeys = []string{"grpc1", "grpc2", "grpc3", "grpc4", "grpc5"}

type websocket struct {
	grpcAddrs []string
}

func NewWebsocket(grpcAddr ...string) *websocket {

	return &websocket{
		grpcAddrs: grpcAddr,
	}
}

func (p *websocket) Handler() *framework.Handler {
	ret := framework.DefaultWsHandler("websocket", p.grpcAddrs)

	ret.Add(
		p.CreateClient,
		p.SendStream,
		p.ReceveResult)

	return ret
}

// CreateClient creates a client
func (p *websocket) CreateClient(sess *framework.Session) error {
	for i, conn := range sess.GrpcConns {
		// 初始化客户端
		// 此处仅模拟使用的是5个相同的grpc服务，实际场景根据业务需求请求相应的grpc 服务
		c := hello.NewStreamServiceClient(conn)
		streamc, err := c.Record(context.Background())
		if err != nil {
			sess.Errorf("CreateClient: fail to call grpc: %s", err.Error())
			return err
		}

		sess.Set(gprcClientKeys[i], streamc)
	}

	return nil
}

func getGrpcClient(sess *framework.Session, gprcClientKey string) hello.StreamService_RecordClient {
	client := sess.MustGet(gprcClientKey)
	if ret, ok := client.(hello.StreamService_RecordClient); ok {
		return ret
	}
	panic("Fail to get Record client")
}

// SendStream
func (p *websocket) SendStream(sess *framework.Session) error {
	// 仅用 grpc1 作演示
	grpc1Client := getGrpcClient(sess, gprcClientKeys[0])

	processMethod := func(data []byte) error {
		grpcRequest := &hello.StreamRequest{
			Pt: &hello.StreamPoint{
				Name:  "gRPC Stream Server: Record",
				Value: getBytesN(1024 * 1024 * 7), // 7M
			},
		}
		ierr := grpc1Client.Send(grpcRequest)
		if ierr != nil {
			sess.Errorf("SendStream: fail to send stream to grpc server: %s", ierr.Error())
		}

		return ierr
	}

	err := framework.StreamForeach(sess, processMethod)
	if err != nil {
		return err
	}

	err = grpc1Client.CloseSend()
	if err != nil {
		sess.Errorf("SendStream: fail to CloseSend of grpc server: %s", err.Error())
		return err
	}

	return nil
}

// ============================================================================

// ReceveResult
func (p *websocket) ReceveResult(sess *framework.Session) error {
	// 仅用 grpc1 作演示
	grpc1Client := getGrpcClient(sess, gprcClientKeys[0])

	grpcResp, err := grpc1Client.CloseAndRecv()
	if err != nil {
		sess.Errorf("ReceveResult: fail to receive response from grpc server: %s", err.Error())
		return err
	}

	err = framework.SendWsResult(sess, fmt.Sprintf("resp: pj.name: %s, len(pt.value): %d", grpcResp.Pt.Name, len(grpcResp.Pt.Value)))
	if err != nil {
		sess.Errorf("ReceveResult: fail to write response to websocket: %s", err.Error())
		return err
	}
	return nil
}

func getBytesN(n int) []byte {
	token := make([]byte, n)
	rand.Read(token)
	return token
}
