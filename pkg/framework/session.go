package framework

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
)

type Session struct {
	Name           string
	ResponseWriter http.ResponseWriter
	Request        *http.Request

	WsConn    *websocket.Conn
	GrpcConns []*grpc.ClientConn
	Ctx       context.Context
	RequestID string
	StartTime time.Time

	keys map[string]interface{}
}

func (p *Session) Set(key string, value interface{}) {
	if p.keys == nil {
		p.keys = make(map[string]interface{})
	}

	p.keys[key] = value
}

func (p *Session) Get(key string) (interface{}, bool) {
	ret, ok := p.keys[key]
	return ret, ok
}

func (p *Session) MustGet(key string) interface{} {
	ret, ok := p.keys[key]
	if ok {
		return ret
	}

	panic("Key '" + key + "' not found")
}

const LogPrefixFormat = "[%s]-[%s]:"

func (p *Session) Info(args ...interface{}) {
	newArgs := append([]interface{}{fmt.Sprintf(LogPrefixFormat, p.Name, p.RequestID)}, args...)
	glog.InfoDepth(1, newArgs...)
}

func (p *Session) Warning(args ...interface{}) {
	newArgs := append([]interface{}{fmt.Sprintf(LogPrefixFormat, p.Name, p.RequestID)}, args...)
	glog.WarningDepth(1, newArgs...)
}

func (p *Session) Error(args ...interface{}) {
	newArgs := append([]interface{}{fmt.Sprintf(LogPrefixFormat, p.Name, p.RequestID)}, args...)
	glog.ErrorDepth(1, newArgs...)
}

func (p *Session) Infof(format string, args ...interface{}) {
	newArgs := append([]interface{}{p.Name, p.RequestID}, args...)
	glog.InfoDepth(1, fmt.Sprintf(LogPrefixFormat+format, newArgs...))
}

func (p *Session) Warningf(format string, args ...interface{}) {
	newArgs := append([]interface{}{p.Name, p.RequestID}, args...)
	glog.WarningDepth(1, fmt.Sprintf(LogPrefixFormat+format, newArgs...))
}

func (p *Session) Errorf(format string, args ...interface{}) {
	newArgs := append([]interface{}{p.Name, p.RequestID}, args...)
	glog.ErrorDepth(1, fmt.Sprintf(LogPrefixFormat+format, newArgs...))
}
