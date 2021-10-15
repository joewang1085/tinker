package framework

import (
	"bytes"
	"fmt"
)

var EOS = []byte{0x45, 0x4f, 0x53}

func IsEOS(data []byte) bool {
	return bytes.Equal(data, EOS)
}

func streamForeach(sess *Session, foreach func(data []byte) error, stopped func() bool) error {
	if stopped == nil {
		stopped = func() bool { return false }
	}
	var err error
	var frame []byte
	for {
		_, frame, err = sess.WsConn.ReadMessage()
		if err != nil {
			sess.Errorf("streamForeach: %v", err.Error())
			err = fmt.Errorf("streamForeach: fail to read stream from client with error: %v", err.Error())
			break
		}

		if IsEOS(frame) || stopped() {
			break
		}

		if len(frame) > 1024*1024*4 {
			sess.Errorf("streamForeach: stream fragmentation overflow: actual(%d) vs max(%d)", len(frame), 1024*1024*4)
			return fmt.Errorf("streamForeach: stream fragmentation overflow")
		}

		err = foreach(frame)
	}

	return err
}

func StreamForeach(sess *Session, foreach func(data []byte) error) error {
	return streamForeach(sess, foreach, nil)
}
