## 开始 ##

    git clone git@192.168.65.10:Y_xConf/go-xConf.git
	
	添加项目到GOPATH中即可集成

## 集成示例 ##

### 配置更新(golang) ###
	// 配置回调handle
	type handle struct {
	}

	func main() {
		defer log.Flush()

		h := handle{}
		
		// 配置更新服务地址,一般以ip:port唯一标识
		localAddr := "192.168.1.1:12345"
		// Create xconf Service
		s, err := xconf.NewService("server1", []string{"v1"}, localAddr)
		if err != nil {
			log.Error("xConf error:", err)
			return
		}
		
		// xConf http服务地址,如果地址为空，则使用"s.xconf.openspeech.cn"进行解析
		xconfHTTPSvrs := []string{"ip4:port4", "ip5:port5"}
		//sdk 初始化
		err = s.Init(xconfHTTPSvrs)
		// Init err最好进行处理，如err不为nil，则异常退出
		if err != nil {
			log.Error("xConf connect to cluster error:", err)
			return
		}

		// 获取配置
		result, err := s.GetCfg(h)
		if err != nil {
			log.Error(err)
			return
		}
		log.Info(result)

		time.Sleep(1000 * time.Second)
	}

	func (h handle) HandlerCfgUpdate(cfgContent map[string][]byte, err error) error {
		log.Info("xConf SDK Handler")
		for k, v := range cfgContent {
			log.Info("k: ", k, "v: ", string(v))
		}
		return nil
	}