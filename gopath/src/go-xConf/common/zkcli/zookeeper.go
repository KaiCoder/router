package zkcli

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/samuel/go-zookeeper/zk"
)

var (
	//
	Connected bool      = false
	Expired   bool      = true
	Monitors  []Monitor = make([]Monitor, 0)
	Exit      chan bool
)

// AddMonitor 。。
func AddMonitor(monitor Monitor) {
	Monitors = append(Monitors, monitor)
}

func RemoveMonitor(monitor Monitor) {
	newMonitors := make([]Monitor, 0)
	for _, m := range Monitors {
		if m != monitor {
			newMonitors = append(newMonitors, m)
		}
	}

	Monitors = newMonitors
}

// 连接到zookeeper服务
func ConnectToZk(zkAddr []string, sessionTimeout time.Duration, provider zk.HostProvider) (*zk.Conn, error) {
	log.Critical("zkaddr: ", zkAddr, " timeout: ", sessionTimeout)

	// add zk address polling
	cOption := zk.WithHostProvider(provider)
	conn, event, err := zk.Connect(zkAddr, sessionTimeout, cOption)
	if err != nil {
		// 连接出错
		log.Error("connectToZk | connect to zookeeper failed: ", err)
		return nil, err
	} else {
		// 连接成功，启动一个goroutine，监视连接状态的变化
		go zkGlobalWatcher(event)
	}

	return conn, err
}

func zkGlobalWatcher(event <-chan zk.Event) {
	Exit = make(chan bool)
	for {
		select {
		case e := <-event:
			log.Debug("zkGlobalWatcher | receive an watcher notification, Type:", e.Type,
				", State:", e.State, ", Path:", e.Path, ", Error:", e.Err)
			switch e.Type {
			case zk.EventSession:
				processSessionEvent(e)
			case zk.EventNodeCreated:
				processNodeCreatedEvent(e)
			case zk.EventNodeDeleted:
				processNodeDeletedEvent(e)
			case zk.EventNodeDataChanged:
				processNodeDataChangedEvent(e)
			case zk.EventNodeChildrenChanged:
				processNodeChildrenChangedEvent(e)
			case zk.EventNotWatching:
			}
		case <-Exit:
			return
		}
	}
}

//处理连接建立事件
func processConnectedEvent() {
	Connected = true
	//首先要判断是由connecting->connected
	//还是由expired->connected

	if !Expired {
		//由connecting->connected，不用处理
		log.Debug("processConnectedEvent | connecting->connected")
		return
	}

	log.Critical("processConnectedEvent | expired->connected")
	//由expired->connected，或者第一次连接到zookeeper
	//需要进行处理，例如创建临时结点，重新注册watcher等等

	//重新注册监视子节点变化的watcher
	for _, m := range Monitors {
		m.OnConnected()
	}

	log.Critical("processConnectedEvent | exit")
	Expired = false
}

func processConnectingEvent() {
	log.Warn("processConnectingEvent | connecting to zookeeper.")
	Connected = false
}

func processDisconnectedEvent() {
	log.Warn("processDisconnectedEvent | disconnect from zookeeper.")
	Connected = false
	for _, m := range Monitors {
		m.OnDisconnected()
	}
}

func processSessionExpireEvent() {
	log.Warn("processSessionExpireEvent | session expired.")
	Connected = false
	Expired = true
	// 测试用
	// children, _, err := ZkConn.Children(ParentNode)
	// if err != nil {
	//	fmt.Println("processConnectedEvent | ZkConn.ChildrenW failed:", err)
	// } else {
	//	fmt.Println("children: ", children)
	// }

	// 貌似不需要重新初始化zkConn
	//ZkConn, _ = reconnectToZk(ZkAddr, SessionTimeout)
}

// 处理session相关的事件
func processSessionEvent(e zk.Event) {
	switch e.State {
	case zk.StateConnecting:
		processConnectingEvent()
	case zk.StateDisconnected:
		processDisconnectedEvent()
	case zk.StateConnected:
		processConnectedEvent()
	case zk.StateExpired:
		processSessionExpireEvent()
	}
}

// 处理节点创建事件
func processNodeCreatedEvent(e zk.Event) {
	log.Debugf("node:%s created.\n", e.Path)
}

//处理节点删除事件
func processNodeDeletedEvent(e zk.Event) {
	log.Debugf("node:%s deleted.\n", e.Path)
}

//处理节点数据变化事件
func processNodeDataChangedEvent(e zk.Event) {
	log.Debugf("node:%s data changed.\n", e.Path)

	for _, m := range Monitors {
		m.OnNodeDataChanged(e.Path)
	}
}

//处理节点子节点变化事件
func processNodeChildrenChangedEvent(e zk.Event) {
	log.Debugf("node:%s children changed.\n", e.Path)

	for _, m := range Monitors {
		m.OnNodeChildrenChanged()
	}
}
