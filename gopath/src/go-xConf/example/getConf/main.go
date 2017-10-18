package main

import (
	log "github.com/cihub/seelog"
	xconf "go-xConf"
	_ "net/http/pprof"
	"time"
)

type handle struct {
}

func main() {
	defer log.Flush()

	h := handle{}
	log.Info("xConf SDK Example")
	s, err := xconf.NewService("its", []string{"v1"}, "127.0.0.12:2121") //sdk 初始化
	if err != nil {
		log.Error("xConf error:", err)
	}

	err = s.Init([]string{"172.16.154.49:2405"}) //连接到配置中心
	if err != nil {
		log.Error("xConf connect to cluster error:", err)
	}

	result, err := s.GetCfg(h)
	if err != nil {
		log.Error(err)
	}

	for k, v := range result {
		log.Info("--------------")
		log.Info("name:", k)
		log.Info("value:", string(v))
		log.Info("--------------")
	}

	for {
		time.Sleep(2 * time.Second)
	}

	err = s.Deregister()
	if err != nil {
		log.Error(err)
	}
	time.Sleep(1 * time.Second)

}

func (h handle) HandlerCfgUpdate(cfgContent map[string][]byte, err error) error {
	log.Info("xConf SDK Handler")
	for k, v := range cfgContent {
		log.Info("k: ", k, "v: ", string(v))
	}
	return nil
}
