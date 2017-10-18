package xconf

import (
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

// ZkMonitor .
type ZkMonitor struct {
	ZkConn      *zk.Conn
	NodeWatched string
	Retry       int64
	FlagSeq     bool // 向zookeeper注册服务时是否启用FlagSequence模式, 默认false
	xMgr        *xManager
	svrHandler  SrvHandler
	cfgHandler  CfgHandler
}

// NewZkMonitor .
func NewZkMonitor() *ZkMonitor {
	return &ZkMonitor{
		Retry:   1,
		FlagSeq: true,
		xMgr:    newXconfManager(),
	}
}

// OnConnected .
func (zm *ZkMonitor) OnConnected() (err error) {
	logger.Info("ZkMonitor::OnConnected enter")

	var i int64
	var tPath string

	if zm.xMgr.getRegFlag() {
		for ; i < zm.Retry; i++ {
			if zm.FlagSeq {
				tPath, err = zm.ZkConn.Create(zm.xMgr.getRegPath(), nil, zk.FlagSequence|zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
			} else {
				tPath, err = zm.ZkConn.Create(zm.xMgr.getRegPath(), nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
			}

			if err != nil {
				logger.Warnf("zk_reginster | Create returned err : %s,retry count:%d wait one second", err.Error(), i)
				time.Sleep(1 * time.Second)
			} else {
				logger.Warnf("zk_reginster | Create success: %s retry count:%d", tPath, i)
				zm.xMgr.setDeregPath(tPath)
				break
			}
		}

		logger.Warnf("zk_reginster | create :%v", tPath)
	}

	if zm.xMgr.getSrvWatchFlag() {
		srvDiscoverMessArray := xMgr.getSrvDiscoverMessArray()
		for _, v := range srvDiscoverMessArray {
			srvNodesReconnectProcess(zm.ZkConn, v.srvPath, v.srvhandler, v.srv, v.tags)
			srvNodesWatch(zm.ZkConn, v.srvPath, v.srvhandler, v.srv, v.tags)
		}
	}

	if zm.xMgr.getCfgWatchFlag() {
		cfgUpdateArray := xMgr.getCfgUpdateMessArray()
		for _, v := range cfgUpdateArray {
			dynamicCfgReconnectProcess(zm.ZkConn, v.cfgPath, v.cfgH, v.srv, v.localAddr)
			dynamicCfgWatch(v.cfgPath, v.cfgH, zm.ZkConn, v.srv, v.localAddr)
		}

	}

	return err
}

// OnDisconnected .
func (zm *ZkMonitor) OnDisconnected() {
	logger.Trace("ZkMonitor::OnDisconnected")

	/*exitChan = make(chan bool, 1)
	exitChan <- true*/
}

// OnConnecting .
func (zm *ZkMonitor) OnConnecting() {
	logger.Trace("ZkMonitor::OnConnecting")
}

// OnNodeChildrenChanged .
func (zm *ZkMonitor) OnNodeChildrenChanged() {
	logger.Trace("ZkMonitor::OnNodeChildrenChanged")
}

// OnSessionExpired .
func (zm *ZkMonitor) OnSessionExpired() {
	logger.Trace("ZkMonitor::OnSessionExpired")
}

// OnNodeDataChanged .
func (zm *ZkMonitor) OnNodeDataChanged(nodePath string) {
	logger.Trace("ZkMonitor::OnNodeDataChanged")
}

// DeletePath .
func (zm *ZkMonitor) DeletePath() {
	tPath := zm.xMgr.getDeregPath()
	if tPath != "" {
		err := zm.ZkConn.Delete(tPath, -1)
		if err != nil {
			logger.Warnf("ZkMonitor::DeletePath | zk delete path:%s failed, err:%s", tPath, err.Error())
		}
		logger.Infof("ZkMonitor::DeletePath | delete path:%s success", tPath)
	} else {
		logger.Infof("ZkMonitor::DeletePath | zm.TPath is nil")
	}
}

func (zm *ZkMonitor) setZkConn(conn *zk.Conn) {
	zm.ZkConn = conn
}
