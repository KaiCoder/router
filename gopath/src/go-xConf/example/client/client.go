// xConf SDK Example
// service discovery feature
package main

import (
	xconf "go-xConf"
	"time"

	"flag"
	xConflog "github.com/cihub/seelog"
)

type handle struct {
}

func main() {
	defer xConflog.Flush()
	xConflog.Info("xConf SDK Example")

	srv := flag.String("s", "its", "srv name")
	addr := flag.String("addr", "127.0.0.1:8080", "service addr")
	srvWatch := flag.String("sw", "its", "what srv name you want to watch")
	flag.Parse()

	s, err := xconf.NewService(*srv, []string{""}, *addr) //sdk 初始化
	if err != nil {
		xConflog.Error("xConf error:", err)
	}

	err = s.Init([]string{"172.16.154.49:2405"}) //连接到配置中心
	if err != nil {
		xConflog.Error("xConf connect to cluster error:", err)
	}

	var h handle
	result, err := s.GetSrvNodes(*srvWatch, []string{""}, h)
	if err != nil {
		xConflog.Info(err)
	}
	xConflog.Info(result)
	for {
		time.Sleep(5 * time.Second)
	}

	s.Fini()
}

func (h handle) HandlerSvrUpdate(msg xconf.SrvMessage, err error) error {
	if err != nil {
		xConflog.Error("xConfSDK HandlerSvrUpdate | get svr message error: ", err)
		return err
	}
	xConflog.Info("xConfSDK HandleUpdate | svr data", msg.SrvAddr)

	return err
}
