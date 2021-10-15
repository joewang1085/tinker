package framework

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/xid"
)

type HttpError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
}

func (p *HttpError) Error() string {
	return fmt.Sprintf("StatusCode: %d; ErrorMessage: %s", p.StatusCode, p.Message)
}

func (p *HttpError) WithMessage(msg string) *HttpError {
	return &HttpError{
		StatusCode: p.StatusCode,
		Message:    msg,
	}
}

var (
	HttpErrorBadRequest = NewHttpError(http.StatusBadRequest, "invalid request")
	HttpErrorServer     = NewHttpError(http.StatusInternalServerError, "internal server error")
)

func NewHttpError(code int, msg string) *HttpError {
	return &HttpError{
		StatusCode: code,
		Message:    msg,
	}
}

func SendHttpError(sess *Session, httpCode int, msg string) error {
	httpErr := HttpError{
		Message: msg,
	}

	return SendHttp(sess, httpCode, &httpErr)
}

func SendHttpResult(sess *Session, result interface{}) error {
	return SendHttp(sess, http.StatusOK, result)
}

func SendHttp(sess *Session, httpCode int, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return SendHttpBinary(sess, httpCode, "applicatoin/json", bytes)
}

func SendHttpBinary(sess *Session, httpCode int, contentType string, data []byte) error {
	rw := sess.ResponseWriter
	header := rw.Header()
	header["Content-Type"] = []string{contentType}

	rw.WriteHeader(httpCode)

	_, err := rw.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func SendHttpChunk(sess *Session, data []byte) error {
	rw := sess.ResponseWriter

	flusher, ok := rw.(http.Flusher)
	if !ok {
		return fmt.Errorf("expected http.ResponseWriter to be an http.Flusher")
	}

	_, err := rw.Write(data)
	if err != nil {
		return err
	}

	flusher.Flush()
	return nil
}

func RequestReadJSON(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}

	decoder := json.NewDecoder(req.Body)
	return decoder.Decode(obj)
}

func WithRequestID() Wrapper {
	return func(sess *Session, action Action) error {
		req := sess.Request
		reqID := req.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = xid.New().String()
		}

		sess.RequestID = reqID
		return action(sess)
	}
}

func WithReplyHttpError() Wrapper {
	return func(sess *Session, action Action) error {
		err := action(sess)
		if err != nil {
			if httpError, ok := err.(*HttpError); ok {
				return SendHttpError(sess, httpError.StatusCode, httpError.Message)
			}

			return SendHttpError(sess, HttpErrorServer.StatusCode, HttpErrorServer.Message)
		}

		return nil
	}
}
