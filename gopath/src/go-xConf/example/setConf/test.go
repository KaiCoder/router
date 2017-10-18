package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	log "github.com/cihub/seelog"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

type ConfigFile struct {
	Name    string `json:"name"`
	Content []byte `json:"content"`
	MD5     string `json:"md5"`
}

type ConfigMap struct {
	Ver       int32        `json:"ver"`
	ConfigMap []ConfigFile `json:"configs"`
}

func main() {
	defer log.Flush()
	conn, _, err := zk.Connect([]string{"172.16.154.49:2181", "172.16.154.49:2182", "172.16.154.50:2181"}, time.Second*5)
	if err != nil {
		log.Error(err)
	}

	config := ConfigFile{}
	content := "hello world dynamic worldsaafsdffd"
	config.Content = []byte(content)
	config.Name = "hello_dynamic.txt"
	md5Hash := md5.Sum([]byte(content))
	config.MD5 = hex.EncodeToString(md5Hash[:])

	configMap := make([]ConfigFile, 0)
	configMap = append(configMap, config)

	t := ConfigMap{}
	t.ConfigMap = configMap
	t.Ver = int32(35)

	data, err := json.Marshal(t)
	if err != nil {
		log.Error(err)
	}

	str := `{"configs":[{"md5":"4937c50bd9ee9983ae2d60e7cf2a37ec","content":"W2NvbW1vbl0KCmtleTEgPSB2YWwxCgpbc2VydmVyXQoKa2V5MiA9IHZhbDIKCnBvcnQgPSAxMjM0NQ==","name":"hello.cfg"}],"ver":2012}`
	for {
		_, err = conn.Set("/xConf/its/config/dynamic", data, -1)
		_, err = conn.Set("/xConf/its/config/dynamic", []byte(str), -1)
	}

	if err != nil {
		log.Error(err)
	}

}
