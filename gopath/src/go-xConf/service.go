package xconf

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"go-xConf/common/zkcli"
	"strings"
	"time"
)

// connect to the configuration hub
func connectToCluster(addrs []string, provider zk.HostProvider) (zkConn *zk.Conn, err error) {
	zkConn, err = zkcli.ConnectToZk(addrs, time.Second*3, provider)
	if err != nil {
		logger.Error("xConf ConnectToCluster | connect to cluster error : ", err)
		return nil, err
	}

	return zkConn, err
}

func getStaticCfg(path string, zkConn *zk.Conn) (cfgFiles []configFile, ver int64, err error) {
	logger.Trace("xConf getStaticCfg")
	cfgMap := configFileArray{}
	data, _, err := zkConn.Get(path)
	if err != nil {
		logger.Error("xConf getStaticCfg | zk get data error:", err)
		return nil, -1, err
	}
	logger.Debug("xconf getStaticCfgzk | zk get data:", string(data))
	err = json.Unmarshal(data, &cfgMap)
	if err != nil {
		logger.Error("xConf getStaticCfg | json unmarshal error:", err)
		return nil, -1, err
	}

	for _, v := range cfgMap.ConfigArray {
		if !isMd5CheckSuccess(v.MD5, v.Content) {
			err = errors.New("MD5 check error")
			break
		}
	}


	return cfgMap.ConfigArray, cfgMap.Ver, err
}

func getDynamicCfg(path string, handler CfgHandler, zkConn *zk.Conn, srv string, localAddr string) (cfgFiles []configFile, ver int64, err error) {
	logger.Trace("xConf getDynamicCfg")
	cfgMap := configFileArray{}
	xMgr := newXconfManager()
	dynamicCfgWatch(path, handler, zkConn, srv, localAddr)

	data, _, err := zkConn.Get(path)
	if err != nil {
		logger.Error("xConf getDynamicCfg | get path content error:", err)
		return nil, -1, err
	}

	if data == nil {
		return nil, -1, errors.New("data is nil")
	}
	err = json.Unmarshal(data, &cfgMap)
	if err != nil {
		logger.Error("xConf getDynamicCfg | unmarshal json error:", err)
	} else {
		result := make(map[string][]byte)
		for _, v := range cfgMap.ConfigArray {
			result[v.Name] = v.Content
		}
		xMgr.setCacheDynamicCfg(result)
	}

	for _, v := range cfgMap.ConfigArray {
		if !isMd5CheckSuccess(v.MD5, v.Content) {
			err = errors.New("MD5 check error")
			break
		}
	}

	logger.Debug("xConf getDynamicCfg | get dynamic Cfg result:", cfgMap.ConfigArray)
	return cfgMap.ConfigArray, cfgMap.Ver, err
}

// dynamic config watch
func dynamicCfgWatch(path string, cfgHandler CfgHandler, zkConn *zk.Conn, srv string, localAddr string) {
	logger.Trace("xConf DynamicCfgWatch")
	cfgMap := configFileArray{}
	httpHandler := newHttpHandler()
	var result map[string][]byte
	result = make(map[string][]byte)

	go func() {
		xMgr := newXconfManager()
		dumpHlr := newDumpHandler(srv)
		for {
			_, _, event, err := zkConn.GetW(path)
			if err != nil {
				logger.Error("xConf DynamicCfgWatch | dynamic config getw error:", err)
				httpHandler.updateFeedback(srv, path, -1, localAddr, 1, err.Error())
				if cfgHandler != nil {
					cfgHandler.HandlerCfgUpdate(nil, err)
				}
				<-event
				continue
			}

			select {
			case events := <-event:
				if events.Err != nil {
					logger.Error("xConf dynamicCfgWatch | event error:", events.Err)
					return
				}

				data, _, err := zkConn.Get(path)

				if data == nil {
					logger.Error("xConf DynamicCfgWatch | dynamic config content is nil")
					httpHandler.updateFeedback(srv, path, -1, localAddr, 1, "data is nil")
					<-event
					continue
				}

				err = json.Unmarshal(data, &cfgMap)
				if err != nil {
					logger.Error("xConf getDynamicCfg | unmarshal json error:", err)
					httpHandler.updateFeedback(srv, path, -1, localAddr, 1, err.Error())
					if cfgHandler != nil {
						cfgHandler.HandlerCfgUpdate(nil, err)
					}
					<-event
					continue
				}

				for _, v := range cfgMap.ConfigArray {
					if !isMd5CheckSuccess(v.MD5, v.Content) {
						err = errors.New("MD5 check error")
						break
					}
				}

				if err != nil {
					logger.Error("xConf getDynamicCfg | config md5 err:", err)
					httphdl.updateFeedback(srv, path, cfgMap.Ver, localAddr, 1, err.Error())
					if cfgHandler != nil {
						cfgHandler.HandlerCfgUpdate(nil, err)
					}
					continue
				}

				if cfgHandler != nil {
					for _, v := range cfgMap.ConfigArray {
						result[v.Name] = v.Content
					}
					cfgHandler.HandlerCfgUpdate(result, err)
					//dump file
					xMgr.setCacheDynamicCfg(result)
					dumpContent := xMgr.getStaticCfg()
					for _, v := range cfgMap.ConfigArray {
						dumpContent = append(dumpContent, v)
					}
					//dumpHlr.writeDumpFile(dumpContent)
					dumpHlr.writeDynamicDumpFile(dumpContent)

					httpHandler.updateFeedback(srv, path, cfgMap.Ver, localAddr, 0, "update success")
					logger.Debug("xConf dynamicCfgWatch | dynamic cfg result:", result)
				}

				//clear the result map
				/*for k := range result{
					delete(result,k)
				}*/

			case <-exitChan:
				logger.Info("xConf dynamicCfgWatch | exit goroutine")
				return
			}

		}
	}()

}

func dynamicCfgReconnectProcess(conn *zk.Conn, path string, h CfgHandler, srv string, localAddr string) {
	cfgMap := configFileArray{}
	httphdl := newHttpHandler()
	xMgr := newXconfManager()
	dumpHdl := newDumpHandler(srv)
	data, _, err := conn.Get(path)
	if err != nil {
		logger.Error("xConf dynamicCfgReconnectProcess | dynamic config getw error:", err)
		httphdl.updateFeedback(srv, path, -1, localAddr, 0, err.Error())
		if h != nil {
			h.HandlerCfgUpdate(nil, err)
		}
		return
	}

	if data == nil {
		logger.Error("xConf dynamicCfgReconnectProcess | dynamic config content is nil")
		httphdl.updateFeedback(srv, path, -1, localAddr, 0, "data is nil")
		return
	}

	err = json.Unmarshal(data, &cfgMap)
	if err != nil {
		logger.Error("xConf dynamicCfgReconnectProcess | unmarshal json error:", err)
		httphdl.updateFeedback(srv, path, -1, localAddr, 0, err.Error())
		if h != nil {
			h.HandlerCfgUpdate(nil, err)
		}
		return
	}
	for _, v := range cfgMap.ConfigArray {
		if !isMd5CheckSuccess(v.MD5, v.Content) {
			err = errors.New("MD5 check error")
			break
		}
	}

	if err != nil {
		logger.Error("xConf dynamicCfgReconnectProcess | config md5 err:", err)
		httphdl.updateFeedback(srv, path, cfgMap.Ver, localAddr, 0, err.Error())
		if h != nil {
			h.HandlerCfgUpdate(nil, err)
		}
		return
	}

	if h != nil {
		var result map[string][]byte
		result = make(map[string][]byte)

		for _, v := range cfgMap.ConfigArray {
			result[v.Name] = v.Content
		}
		result, needNotified := xMgr.getCacheDynamicCfg(result)
		if needNotified {
			h.HandlerCfgUpdate(result, err)
			httphdl.updateFeedback(srv, path, cfgMap.Ver, localAddr, 1, "update success")

			dumpContent := xMgr.getStaticCfg()
			for _, v := range cfgMap.ConfigArray {
				dumpContent = append(dumpContent, v)
			}
			dumpHdl.writeDynamicDumpFile(dumpContent)

		}

	}

}

func srvNodesWatch(c *zk.Conn, path string, svrHandler SrvHandler, srv string, tags []string) (result []string, err error) {
	logger.Trace("xConf srvNodesWatch")
	var mess SrvMessage
	mess.Srv = srv
	mess.Tags = tags
	xMgr := newXconfManager()

	go func() {
		for {
			_, _, event, err := c.ChildrenW(path)
			if err != nil {
				logger.Error("xConf srvNodesWatch | get childrenw error:", err)
				time.Sleep(4 * time.Second)
			}

			select {
			case events := <-event:
				if events.Err != nil {
					logger.Error(events.Err)
					return
				}
				nodes, _, err := c.Children(path)

				if err != nil {
					logger.Error("xConf srvNodesWatch | get child error:", err)
					if svrHandler != nil {
						svrHandler.HandlerSvrUpdate(mess, err)
					}
					time.Sleep(time.Second * 1)
					continue
				}

				svrAddr, err := addrDeserialization(nodes)
				if err != nil {
					logger.Error("xConf srvNodesWatch | nodes deserialization error:", err, nodes)
					if svrHandler != nil {
						svrHandler.HandlerSvrUpdate(mess, err)
					}
					time.Sleep(time.Second * 1)
					continue
				}

				mess.SrvAddr = svrAddr

				if svrHandler != nil {
					svrHandler.HandlerSvrUpdate(mess, err)
					xMgr.setCacheAddress(mess.SrvAddr)
					logger.Info("xConf srvNodesWatch | svr mess:", mess)
				}

			case <-exitChan:
				logger.Info("xConf srvNodesWatch | exit goroutine")
				return
			}
		}
	}()

	nodes, _, err := c.Children(path)
	if err != nil {
		logger.Error("xConf srvNodesWatch | get children error:", err)
	}
	result, err = addrDeserialization(nodes)
	xMgr.setCacheAddress(result)
	logger.Info("xConf srvNodesWatch | svr node result:", result)

	return result, err
}

func srvNodesReconnectProcess(conn *zk.Conn, path string, h SrvHandler, srv string, tags []string) {
	xMgr := newXconfManager()
	nodes, _, err := conn.Children(path)
	if err != nil {
		logger.Error("xConf srvNodesReconnectProcess | get children error:", err)
		return
	}
	result, err := addrDeserialization(nodes)
	result, needNotified := xMgr.getCacheAddress(result)

	if needNotified {
		mess := SrvMessage{
			Srv:     srv,
			Tags:    tags,
			SrvAddr: result,
		}
		h.HandlerSvrUpdate(mess, err)
	}
}

// addrDeserialization
func addrDeserialization(nodes []string) (nodeValues []string, err error) {
	nodeValues = make([]string, 0)

	for _, node := range nodes {
		if len(node)-10 <= 0 {
			logger.Warn("address format is not right,not be serialization,addr:", node)
			err = errors.New(fmt.Sprintf("address format is not right,not be serialization,addr:%s", node))
			continue
		}

		node = node[:len(node)-10]
		nodeValues = append(nodeValues, node)
	}
	return nodeValues, nil
}

// isMd5CheckSuccess
func isMd5CheckSuccess(md5SumStr string, content []byte) bool {
	md5Hase := md5.Sum(content)
	md5Str := hex.EncodeToString(md5Hase[:])
	return strings.EqualFold(md5SumStr, md5Str)
}
