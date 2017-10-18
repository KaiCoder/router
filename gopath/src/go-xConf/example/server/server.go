// xConf server SDK example
// Register & GetCfg
package main

import (
	"flag"
	xConflog "github.com/cihub/seelog"
	xconf "go-xConf"
	"net/http"
	_ "net/http/pprof"
	"time"
	"os"
)

type handle struct {
}

func main() {
	defer xConflog.Flush()

	go func() {
		xConflog.Info(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	srv := flag.String("s", "its", "srv name")
	addr := flag.String("addr", "127.0.0.1:8080", "service addr")
	flag.Parse()

	xConflog.Info("xConf SDK Example")
	s, err := xconf.NewService(*srv, []string{""}, *addr) //sdk 初始化
	if err != nil {
		xConflog.Error("xConf error:", err)
	}

	err = s.Init([]string{"172.16.154.49:2405"}) //连接到配置中心
	//err = s.Init(nil) //连接到配置中心
	if err != nil {
		xConflog.Error("xConf connect to cluster error:", err)
		os.Exit(-1)
	}

	s.Register() //registration service

	h := handle{}
	//
	result, err := s.GetCfg(h)
	if err != nil {
		xConflog.Error(err)
	}

	for k, v := range result {
		xConflog.Info("--------------------")
		xConflog.Info("k: ", k, "v: ", string(v))
	}

	for {
		time.Sleep(3 * time.Second)
	}

	err = s.Deregister()
	if err != nil {
		xConflog.Error(err)
	}

	s.Fini()
}

func (h handle) HandlerCfgUpdate(cfgContent map[string][]byte, err error) error {
	xConflog.Info("xConf SDK Handler")
	for k, v := range cfgContent {
		xConflog.Info("k: ", k, "v: ", string(v))
	}

	if err != nil {
		xConflog.Info("xConf SDK Handler error:", err)
	}

	return err
}
