package api

import (
	"fmt"
	"net/http"

	"tinker/pkg/api/httpcase"
	"tinker/pkg/api/websocket"

	"github.com/golang/glog"
	"golang.org/x/sync/errgroup"
)

func Serve() (err error) {

	grpcAddr1 := "127.0.0.1:8686"
	grpcAddr2 := "127.0.0.1:8686"
	grpcAddr3 := "127.0.0.1:8686"
	grpcAddr4 := "127.0.0.1:8686"
	grpcAddr5 := "127.0.0.1:8686"

	// kong auth
	var errGroup errgroup.Group
	errGroup.Go(func() error {
		httpCon := httpcase.NewHttpCase(grpcAddr1, grpcAddr2)
		http.Handle("/httpcase", httpCon.Handler())

		streamCon := websocket.NewWebsocket(grpcAddr1, grpcAddr2, grpcAddr3, grpcAddr4, grpcAddr5)
		http.Handle("/websocket", streamCon.Handler())

		if err := http.ListenAndServe(fmt.Sprintf(":%d", 8585), nil); err != nil {
			glog.Errorf("api exit with error: %s", err.Error())
		}

		return err
	})

	return errGroup.Wait()
}
