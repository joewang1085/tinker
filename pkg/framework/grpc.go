package framework

import (
	"time"

	"google.golang.org/grpc"
)

// WithGrpc returns a Wrapper creating and releasing grpc connection.
// The grpc connection is set in Session.GrpcConns
func WithGrpc(targets []string) Wrapper {
	return func(sess *Session, action Action) error {
		for _, target := range targets {
			grpcConn, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*10))
			if err != nil {
				sess.Errorf("WithGrpc: fail to dial grpc endpoint '%s': %s", target, err.Error())
				return err
			}
			defer func() {
				grpcConn.Close()
				sess.Infof("WithGrpc: grpc connection closed")
			}()

			sess.Infof("WithGrpc: connected to grpc endpoint '%s'", target)
			sess.GrpcConns = append(sess.GrpcConns, grpcConn)
		}

		return action(sess)
	}
}
