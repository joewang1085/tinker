package main

import (
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"tinker/pkg/api"
)

var (
	appCmd = &cobra.Command{
		Short: "start tinker server",
		RunE:  execute,
	}
)

func main() {
	if err := appCmd.Execute(); err != nil {
		glog.Error("exit with:", err.Error())
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) (err error) {
	return api.Serve()
}
