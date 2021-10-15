package framework

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"
)

type Handler struct {
	Name string

	wrappers []Wrapper
	actions  []Action

	OnError func(*Session, error)
	OnPanic func(*Session, interface{})
}

func LogError(sess *Session, err error) {
	sess.Errorf("Unhandled error: %s", err)
}

func LogPanic(sess *Session, pani interface{}) {
	sess.Errorf("Recover from panic: %v", pani)
}

func (p *Handler) Use(wrappers ...Wrapper) {
	p.wrappers = append(p.wrappers, wrappers...)
}

func (p *Handler) UseFirst(wrappers ...Wrapper) {
	p.wrappers = append(wrappers, p.wrappers...)
}

func (p *Handler) Add(actions ...Action) {
	p.actions = append(p.actions, actions...)
}

func DefaultWsHandler(name string, grpcAddr []string) *Handler {
	ret := &Handler{
		Name:    name,
		OnError: LogError,
		OnPanic: LogPanic,
	}
	ret.Use(WithRequestID(), WithWebsocket(), WithReplyWsError(), WithGrpc(grpcAddr))
	return ret
}

func DefaultHttpHandler(name string, grpcAddr []string) *Handler {
	ret := &Handler{
		Name:    name,
		OnError: LogError,
		OnPanic: LogPanic,
	}
	ret.Use(WithRequestID(), WithReplyHttpError(), WithGrpc(grpcAddr))
	return ret
}

// should create grpc conn by self, because session.GrpcConn is nil
func EmptyGrpcConnHttpHandler(name string) *Handler {
	ret := &Handler{
		Name:    name,
		OnError: LogError,
		OnPanic: LogPanic,
	}
	ret.Use(WithRequestID(), WithReplyHttpError())
	return ret
}

func (p *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var err error
	sess := new(Session)
	sess.Name = p.Name
	sess.Ctx = context.Background()
	sess.ResponseWriter = rw
	sess.Request = req
	sess.StartTime = time.Now().UTC()

	defer func() {
		latency := time.Since(sess.StartTime).Seconds()
		sess.Infof("requestId:%s latency is %d", sess.RequestID, latency)

		if perr := recover(); perr != nil {
			debug.PrintStack()
			p.OnPanic(sess, perr)
			return
		}

		if err != nil {
			p.OnError(sess, err)
			return
		}
	}()

	fin := make(chan int)
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		mainAction := Seq(p.actions...).WithWrappers(p.wrappers...)
		err = mainAction(sess)
		fin <- 1
	}()

	select {
	case <-fin:
	case <-ticker.C:
		if sess.WsConn != nil {
			sess.Errorf("Session timeout")
			err = SendWsError(sess, CodeServerError, "Session timeout")
		} else {
			sess.Errorf("Session timeout")
			err = SendHttpError(sess, http.StatusInternalServerError, "Session timeout")
		}
	}

}
