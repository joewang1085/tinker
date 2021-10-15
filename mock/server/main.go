package main

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"net"

	"google.golang.org/protobuf/types/known/anypb"

	hello "tinker/mock/pb/hello"

	// "github.com/golang/protobuf/ptypes"
	// any "github.com/golang/protobuf/ptypes/any"
	// "google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"
	// timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	anyUrlPrefix string = "type.googleapis.com/"
)

type Server struct {
}

// 实现Love服务接口
func (g *Server) Greet(ctx context.Context, request *hello.GreetRequest) (*hello.GreetResponse, error) {
	resp := &hello.GreetResponse{}
	switch request.GetSaying() {
	case "Hi":
		resp.Acking = "Hi " + hello.Name_name[int32(request.GetPerson())]
	case "Hello":
		resp.Acking = "Hello" + hello.Name_name[int32(request.GetPerson())]
	case "Nice meeting you":
		resp.Acking = "you too"
	default:
		resp.Acking = "Hi " + hello.Name_name[int32(request.GetPerson())]
	}

	resp.Name = hello.Name_Robot
	resp.Time = timestamppb.Now()

	//  >>>>>>>>>>>> any <<<<<<<<<<<<<<<<<
	// data := &any.Any{Value: []byte("How are you?")}
	// packedReply, err := ptypes.MarshalAny(data)
	// if err != nil {
	// 	panic(err)
	// }
	// resp.Data = packedReply

	// question
	// question := &hello.Question{
	// 	Question: "How are you?",
	// }
	// err := resp.GetData().MarshalFrom(question)   // ==> 不明白为什么报空指针错误
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // anypb.MarshalFrom()
	// func MarshalFrom(dst *Any, src proto.Message, opts proto.MarshalOptions) error {
	// 	const urlPrefix = "type.googleapis.com/"
	// 	if src == nil {
	// 		return protoimpl.X.NewError("invalid nil source message")
	// 	}
	// 	b, err := opts.Marshal(src)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	dst.TypeUrl = urlPrefix + string(src.ProtoReflect().Descriptor().FullName())
	// 	dst.Value = b
	// 	return nil
	// }

	question := &hello.Question{
		Question: "How are you?",
	}
	data, err := proto.Marshal(question)
	if err != nil {
		panic(err)
	}
	resp.Data = &anypb.Any{
		TypeUrl: anyUrlPrefix + "hello.Question", // 约定全局唯一，默认 `type.googleapis.com/packagename.messagename`
		Value:   data,
	}

	// >>>>>>>>>>>>>>>>> oneof <<<<<<<<<<<<<<<<<<<<<
	resp.Topic = &hello.Topic{
		Content: &hello.Topic_Question{
			Question: question,
		},
	}

	// >>>>>>>>>>>>>>>>>>>> map <<<<<<<<<<<<<<<
	resp.AllTopics = make(map[string]*hello.Topic)

	resp.AllTopics["question1"] = &hello.Topic{
		Content: &hello.Topic_Question{
			Question: &hello.Question{
				Question: "How are you?",
			},
		},
	}

	resp.AllTopics["question2"] = &hello.Topic{
		Content: &hello.Topic_Question{
			Question: &hello.Question{
				Question: "How is going?",
			},
		},
	}

	resp.AllTopics["action1"] = &hello.Topic{
		Content: &hello.Topic_Action{
			Action: "Wave!",
		},
	}

	resp.AllTopics["action2"] = &hello.Topic{
		Content: &hello.Topic_Action{
			Action: "Nod!",
		},
	}

	resp.AllTopics["action3"] = &hello.Topic{
		Content: &hello.Topic_Action{
			Action: "Shake hands!",
		},
	}

	return resp, nil
}

func main() {
	// 监听8888端口
	listen, err := net.Listen("tcp", ":8686")
	if err != nil {
		log.Fatal(err)
	}

	// 实例化grpc server
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*8), // server max recv msg , 默认 4 m
		grpc.MaxSendMsgSize(1024*1024*8), // server max send msg , 默认 不限制
	)

	// 注册Love服务
	hello.RegisterGreetingServer(s, new(Server))
	hello.RegisterStreamServiceServer(s, new(Server))

	log.Println("Listen on 127.0.0.1:8686...")
	s.Serve(listen)
}

func (s *Server) List(r *hello.StreamRequest, stream hello.StreamService_ListServer) error {
	for n := int32(0); n <= 6; n++ {
		err := stream.Send(&hello.StreamResponse{
			Pt: &hello.StreamPoint{
				Name:  r.Pt.Name,
				Value: getBytesN(1024 * 1014 * 6), // 6m
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Record(stream hello.StreamService_RecordServer) error {
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&hello.StreamResponse{
				Pt: &hello.StreamPoint{
					Name: "gRPC Stream Server: Record",
				},
			})
		}
		if err != nil {
			return err
		}

		log.Printf("stream.Recv pt.name: %s, len(pt.value): %d", r.Pt.Name, len(r.Pt.Value))
	}
}

func (s *Server) Route(stream hello.StreamService_RouteServer) error {
	n := 0
	for {
		err := stream.Send(&hello.StreamResponse{
			Pt: &hello.StreamPoint{
				Name:  "gPRC Stream Client: Route",
				Value: getBytesN(1024 * 1024 * 6), // 6m
			},
		})
		if err != nil {
			return err
		}

		r, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		n++

		log.Printf("stream.Recv pt.name: %s, len(pt.value): %d", r.Pt.Name, len(r.Pt.Value))
	}
}

func (s *Server) Route2(stream hello.StreamService_Route2Server) error {
	ret := make(chan error)
	recv := make(chan int)

	go func() {
		for {

			<-recv
			for n := 0; n <= 6; n++ {
				err := stream.Send(&hello.StreamResponse{ // send 只能发送一次，阻塞直到 recv，才能第二次 send.
					Pt: &hello.StreamPoint{
						Name:  "gPRC Stream Client: Route2",
						Value: getBytesN(1024 * 1024 * 6), // 6m
					},
				})
				if err != nil {
					ret <- err
				}

			}
			r, err := stream.Recv()
			if err == io.EOF {
				ret <- nil
			}
			if err != nil {
				ret <- err
			}

			log.Printf("stream.Recv pt.name: %s, len(pt.value): %d", r.Pt.Name, len(r.Pt.Value))
		}
	}()

	go func() {
		for {
			r, err := stream.Recv()
			if err == io.EOF {
				ret <- nil
			}
			if err != nil {
				ret <- err
			}

			recv <- 1

			log.Printf("stream.Recv pt.name: %s, len(pt.value): %d", r.Pt.Name, len(r.Pt.Value))
		}
	}()

	return <-ret
}

func getBytesN(n int) []byte {
	token := make([]byte, n)
	rand.Read(token)
	return token
}
