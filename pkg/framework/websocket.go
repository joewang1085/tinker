package framework

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	TypeSuccess = "success"
	TypeError   = "error"
)

type WsResponse struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id"`
	Data      interface{} `json:"data"`
}

const (
	CodeClientError = 2400
	CodeServerError = 2500
)

var (
	WsErrorClient = NewWsError(CodeClientError, "client error")
	WsErrorServer = NewWsError(CodeServerError, "internal server error")
)

type WsError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewWsError(code int, msg string) *WsError {
	return &WsError{
		Code:    code,
		Message: msg,
	}
}

func (p *WsError) Error() string {
	return fmt.Sprintf("ErrorCode: %d; ErrorMessage: %s", p.Code, p.Message)
}

func (p *WsError) WithMessage(msg string) *WsError {
	return &WsError{
		Code:    p.Code,
		Message: msg,
	}
}

func SendWsError(sess *Session, code int, msg string) error {
	resp := WsResponse{
		Type:      TypeError,
		RequestID: sess.RequestID,
		Data: &WsError{
			Code:    code,
			Message: msg,
		},
	}

	err := sess.WsConn.WriteJSON(&resp)
	if err != nil {
		return err
	}

	return nil
}

func SendWsResult(sess *Session, result interface{}) error {
	resp := WsResponse{
		Type:      TypeSuccess,
		RequestID: sess.RequestID,
		Data:      result,
	}

	err := sess.WsConn.WriteJSON(&resp)
	if err != nil {
		return err
	}

	return nil
}

func WithWebsocket() Wrapper {
	return func(sess *Session, action Action) error {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(req *http.Request) bool {
				// cors should be handled by api gateway
				// bypass origin check
				return true
			},
		}

		wsConn, err := upgrader.Upgrade(sess.ResponseWriter, sess.Request, nil)
		if err != nil {
			sess.Errorf("WithWebsocket: failed to upgrade to websocket: %s", err.Error())
			return err
		}
		defer func() {
			// send close control message before close the underlying connection
			derr := wsConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1000, "done"), time.Now().Add(time.Second*5))
			if derr != nil {
				sess.Warningf("Fail to send close message: %v", derr)
			}
			// Note(yiwei.gu): Thanks to the mysterious connection closing control
			// by gorilla/websocket, we never know when the underlying connection
			// is closed exactly. So we decided to add an extra delay(5s) before
			// truly closing the connection. Hope they'll fix it one day :)
			// refer to https://github.com/gorilla/websocket/pull/487
			time.Sleep(time.Duration(5) * time.Second)
			// close connection
			wsConn.Close()
			sess.Infof("WithWebsocket: websocket closed")
		}()
		sess.Infof("WithWebsocket: upgrade to websocket")
		sess.WsConn = wsConn

		return action(sess)
	}
}

func WithReplyWsError() Wrapper {
	return func(sess *Session, action Action) error {
		err := action(sess)
		if err != nil {
			if appError, ok := err.(*WsError); ok {
				return SendWsError(sess, appError.Code, appError.Message)
			}

			return SendWsError(sess, CodeServerError, "internal server error")
		}

		return nil
	}
}
