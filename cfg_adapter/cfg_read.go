package cfg_adapter

import (
	"log"

	"github.com/go-ini/ini"
)

var (
	GOSERVER        map[string]string = make(map[string]string)
	MQSERVER		 map[string]string = make(map[string]string)
	CONNTECTPOOL   map[string]string = make(map[string]string)
)

func init() {
	//读取配置文件
	cfg, err := ini.Load("router.cfg")
	if err != nil {
		log.Fatal(err.Error())
	}
	cfg.BlockMode = false
	//===================================================================
	//初始化LOCAL
	func() {
		section, err := cfg.GetSection("find_server")
		if err != nil {
			log.Fatal(err.Error())
		}
		GOSERVER = section.KeysHash()
	}()
	//===================================================================
	//初始化MQ
	func() {
		section, err := cfg.GetSection("mq_server")
		if err != nil {
			log.Fatal(err.Error())
		}
		MQSERVER = section.KeysHash()
	}()

	func() {
		section, err := cfg.GetSection("connect")
		if err != nil {
			log.Fatal(err.Error())
		}
		CONNTECTPOOL = section.KeysHash()
	}()
}
