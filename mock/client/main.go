package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	hello "tinker/mock/pb/hello"

	// "google.golang.org/grpc"
	// proto "google.golang.org/protobuf/proto"
	// anypb "google.golang.org/protobuf/types/known/anypb"
	// "github.com/golang/protobuf/ptypes"
	// any "github.com/golang/protobuf/ptypes/any"
	// timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/golang/protobuf/ptypes"
	any "github.com/golang/protobuf/ptypes/any"
	"google.golang.org/grpc"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	anyUrlPrefix string = "type.googleapis.com/"
)

func main() {
	// 连接grpc服务
	conn, err := grpc.Dial(":8686",
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024*1024*8), // client max recv msg , 默认 4m
			grpc.MaxCallSendMsgSize(1024*1024*8), // client max send msg , 默认不限制
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	// 很关键
	defer conn.Close()

	// 初始化客户端
	c := hello.NewGreetingClient(conn)
	streamc := hello.NewStreamServiceClient(conn)

	// 发起请求
	request := &hello.GreetRequest{
		Saying: "hello",
		Person: hello.Name_Joe,
		Time:   timestamppb.Now(),
		Fruit:  hello.GreetRequest_apple,
	}

	response, err := c.Greet(context.Background(), request)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("data", response.Data)
	log.Println(request.Time.GetSeconds(), hello.Name_name[int32(request.GetPerson())]+": ", `"`+request.GetSaying()+`"`)
	log.Println(response.Time.GetSeconds(), hello.Name_name[int32(response.GetName())]+": ", `"`+response.GetAcking()+`"`)
	// q := &hello.Question{}
	switch strings.TrimPrefix(response.GetData().GetTypeUrl(), anyUrlPrefix) {
	case "google.protobuf.Any":
		a := &any.Any{}
		err = ptypes.UnmarshalAny(response.Data, a)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("any", a)
	case "hello.Question":
		q := &hello.Question{}
		err = ptypes.UnmarshalAny(response.Data, q)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(response.Time.GetSeconds(), hello.Name_name[int32(response.GetName())]+": ", `"`+q.Question+`"`)
	case "hello.Topic":
		a := &hello.Topic{}
		err = ptypes.UnmarshalAny(response.Data, a)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(response.Time.GetSeconds(), hello.Name_name[int32(response.GetName())]+": ", a)
	}

	switch response.GetTopic().GetContent().(type) {
	case *(hello.Topic_Question):
		log.Println("response type question, content:", response.GetTopic().GetContent().(*hello.Topic_Question))
	case *(hello.Topic_Action):
		log.Println("response type action, content:", response.GetTopic().GetContent().(*hello.Topic_Action))
	}

	log.Println("print all topic:")
	for k, v := range response.AllTopics {
		switch v.GetContent().(type) {
		case *(hello.Topic_Question):
			log.Println("question:", k, ", content:", v.GetContent().(*hello.Topic_Question))
		case *(hello.Topic_Action):
			log.Println("question:", k, ", content:", v.GetContent().(*hello.Topic_Action))
		}
	}

	// note :
	// request.ProtoReflect()
	// protoreflect.Message()
	// protoreflect.ProtoMessage()
	// proto.Message

	log.Println("Server stream test: List")
	err = printLists(streamc, &hello.StreamRequest{
		Pt: &hello.StreamPoint{
			Name: "gRPC Stream Server: List",
		},
	})
	if err != nil {
		panic(err)
	}

	log.Println("Server stream test: Record")
	err = printRecord(streamc, &hello.StreamRequest{
		Pt: &hello.StreamPoint{
			Name:  "gRPC Stream Server: Record",
			Value: getBytesN(1024 * 1024 * 7), // 7M
		},
	})
	if err != nil {
		panic(err)
	}

	log.Println("Server stream test: Route")
	err = printRoute(streamc, &hello.StreamRequest{
		Pt: &hello.StreamPoint{
			Name:  "gRPC Stream Server: Route",
			Value: getBytesN(1024 * 1024 * 7), // 7M
		},
	})
	if err != nil {
		panic(err)
	}

	log.Println("Server stream test: Route2")
	err = printRoute2(streamc, &hello.StreamRequest{
		Pt: &hello.StreamPoint{
			Name:  "gRPC Stream Server: Route2",
			Value: getBytesN(1024 * 1024 * 7), // 7M
		},
	})
	if err != nil {
		panic(err)
	}

}

func printLists(client hello.StreamServiceClient, r *hello.StreamRequest) error {
	stream, err := client.List(context.Background(), r)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		log.Printf("resp: pj.name: %s, len(pt.value): %d", resp.Pt.Name, len(resp.Pt.Value))
	}

	return nil
}

func printRecord(client hello.StreamServiceClient, r *hello.StreamRequest) error {
	stream, err := client.Record(context.Background())
	if err != nil {
		return err
	}

	for n := 0; n <= 6; n++ {
		err := stream.Send(r)
		if err != nil {
			return err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	log.Printf("resp: pj.name: %s, len(pt.value): %d", resp.Pt.Name, len(resp.Pt.Value))

	return nil
}

func getBytesN(n int) []byte {
	token := make([]byte, n)
	rand.Read(token)
	return token
}

func printRoute(client hello.StreamServiceClient, r *hello.StreamRequest) error {
	stream, err := client.Route(context.Background())
	if err != nil {
		return err
	}

	for n := 0; n <= 6; n++ {
		err = stream.Send(r)
		if err != nil {
			return err
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		log.Printf("resp: pj.name: %s, len(pt.value): %d", resp.Pt.Name, len(resp.Pt.Value))
	}

	stream.CloseSend()

	return nil
}

func printRoute2(client hello.StreamServiceClient, r *hello.StreamRequest) error {
	stream, err := client.Route(context.Background())
	if err != nil {
		return err
	}

	ret := make(chan error)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				ret <- err
			}

			log.Printf("resp: pj.name: %s, len(pt.value): %d", resp.Pt.Name, len(resp.Pt.Value))
		}
	}()

	go func() {

		c := time.Tick(5 * time.Second)
		for range c {
			err = stream.Send(r)
			if err != nil {
				ret <- err
			}

			time.Sleep(10 * time.Second)
			ret <- nil
		}

	}()

	<-ret
	stream.CloseSend()

	return nil
}
