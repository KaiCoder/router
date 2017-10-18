package collector_adapter

import (
	"time"
	cfg "../cfg_adapter"
	net_int "../gen-go/net_interface"

	xconf "go-xConf"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/cihub/seelog"
	"sync"
	"encoding/json"
	"strconv"
)

type collector *net_int.CountServiceClient
type collector_set map[collector]bool

type handle struct {
}

//非汇总json object
type CountStruct struct {
	Version   string `json:"ver"`
	AppidName string `json:"appid"`
	Uid string `json:"uid"`
	Channels []struct {
		ChannelName string `json:"sub"`
		Funcs []struct {
			FuncName  string `json:"func"`
			Timestamp int `json:"time"`
			Count     int `json:"cnt"`
		} `json:"funcs"`
	} `json:"subs"`
}

//汇总json object
type IncrementalStruct []struct {
	AppidName string `json:"appid_name"`
	Channels []struct {
		ChannelName string `json:"channel_name"`
		Funcs []struct {
			FuncName string `json:"func_name"`
			Limit struct {
				Timestamp int `json:"timestamp"`
				Count     int `json:"count"`
			} `json:"limit"`
		} `json:"funcs"`
	} `json:"channels"`
}

//server status
var v_status map[collector]int32
var status_mutex sync.Mutex

//map key:server-addr value:collector sets,connect pool
var v_serverMap map[string]collector_set
var mutex sync.Mutex

//consistent hash
var hash *HashRing

//connect poolsize
var poolsize int
var TIMEOUT int


func (h handle) HandlerSvrUpdate(msg xconf.SrvMessage, err error) error {
	if err != nil {
		seelog.Errorf("xConf HandlerSvrUpdate callback is error:: ", err)
		return err
	}
	seelog.Infof("xConf HandlerSvrUpdate callback: ",msg.SrvAddr)

	set := map[string]bool{}

	mutex.Lock()
	defer mutex.Unlock()

	//add new online server
	for _, addr := range msg.SrvAddr {
		if _, ok := v_serverMap[addr]; !ok { //no found, add it!
			server, e := newMultiServer(addr)
			if e == nil {
				v_serverMap[addr] = server
				hash.AddNode(addr, 1)
			}
		}
		set[addr] = true
	}

	//remove offline server
	for addr,_ := range v_serverMap {
		if _,ok := set[addr]; !ok {
			hash.RemoveNode(addr)
			delete(v_serverMap, addr)
		}
	}

	return err
}

func init() {
	v_status = make(map[collector]int32)
	v_serverMap = make(map[string]collector_set)
	virtualSpots := 100
	hash = NewHashRing(virtualSpots)
	if v, err := strconv.Atoi(cfg.CONNTECTPOOL["poolsize"]); err == nil {
		poolsize = v
	} else {
		poolsize = 2
	}

	if v, err := strconv.Atoi(cfg.CONNTECTPOOL["timeout"]); err == nil {
		TIMEOUT = v
	} else {
		TIMEOUT = 5
	}

	go func() {
		s, err := xconf.NewService(cfg.GOSERVER["srv"], []string{""}, cfg.GOSERVER["local_addr"]) //sdk 初始化
		if err != nil {
			seelog.Errorf("xConf error:", err)
		}

		err = s.Init([]string{cfg.GOSERVER["remote_addr"]})
		if err != nil {
			seelog.Errorf("xConf connect to cluster error:", err)
		}

		var h handle
		result, err := s.GetSrvNodes(cfg.GOSERVER["watch_name"], []string{cfg.GOSERVER["tag_name"]}, h)
		if err != nil {
			seelog.Errorf("xConf GetSrvNodes error:", err)
		} else {
			seelog.Infof("xConf GetSrvNodes return string: ", result)

			// 写入
			for _,addr := range result {
				if _, ok := v_serverMap[addr]; !ok {
					server, e := newMultiServer(addr)
					if e == nil {
						v_serverMap[addr] = server
						hash.AddNode(addr, 1)
					}
				}
			}
		}

		for {
			time.Sleep(5 * time.Second)
		}

		s.Fini()
	}()
}

func newServer(addr string) (collector, error) { //连接服务
	transportFactory := thrift.NewTBufferedTransportFactory(10240)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	//transport, err := thrift.NewTSocket(addr)
	transport, err := thrift.NewTSocketTimeout(addr, time.Duration(TIMEOUT)*time.Second)
	if err != nil {
		seelog.Errorf("new socket addr<%s> error:%v", addr, err)
		return nil, err
	} else {
		seelog.Infof("connect to server %s success", addr)
	}

	transport.SetTimeout(time.Duration(TIMEOUT)*time.Second)

	useTransport, err := transportFactory.GetTransport(transport)
	server := net_int.NewCountServiceClientFactory(useTransport, protocolFactory)
	if err := transport.Open(); err != nil {
		seelog.Errorf("socket open addr<%s> error:%v", addr, err)
		return nil, err
	}

	return server, nil
}

func closeServer(server collector) error { //关闭连接
	server.Transport.Close()
	return nil
}

func newMultiServer(addr string) (collector_set, error) {
	seelog.Infof("poolsize=%d", poolsize)
	collectors := make(collector_set)
	//wait for an second ,wait for metering_sea ready
	time.Sleep(1*time.Second)

	for i := 0; i < poolsize; i++ {
		server, err := newServer(addr)
		if err == nil {
			collectors[server] = true
			v_status[server] = 0
		}
	}

	return collectors, nil
}

func addServer(addr string) { //重连时调用
	server,err := newServer(addr)
	if err == nil {
		mutex.Lock()
		if _,ok := v_serverMap[addr]; ok {
			v_serverMap[addr][server] = true
		}
		mutex.Unlock()

		status_mutex.Lock()
		v_status[server] = 0
		status_mutex.Unlock()
	}
}

func deleteServer(server collector, addr string) {
	mutex.Lock()
	defer mutex.Unlock()

	if v,ok := v_serverMap[addr]; ok {
		if _,o := v[server]; o {
			delete(v_serverMap[addr], server)
		}
	}
}

func ParseMessage(msg []byte) (appid string, err error) {  //消息解析
	var m CountStruct //非汇总数据
	err = json.Unmarshal(msg, &m) //将msg作为一个JSON进行解析，解析后的数据存储在参数m中
	if err != nil {
		return "", err
	}

	return m.AppidName, nil
}

func ParseMultiMessage(msg []byte)(map[string][]byte, error) {
	var m IncrementalStruct //汇总数据
	err := json.Unmarshal(msg, &m)
	if err != nil {
		return nil, err
	}

	appids := make(map[string][]byte)  
	for _,k := range m {
		if appid,err := json.Marshal(k); err==nil {
			appids[k.AppidName] = appid
		}
	}
	return appids, nil
}

func GetServerByAppID(appID string) (*net_int.CountServiceClient, string) {
	mutex.Lock()
	defer mutex.Unlock()
	addr := hash.GetNode(appID)  //根据appID获取节点

	if v, ok := v_serverMap[addr]; ok {  //v_serverMap: map[string]collector_set    collector_set: map[collector]bool
		for i,j := range v {
			if j {
				v[i] = false
				return i, addr
			}
		}

		return nil, "flag"
	}

	return nil, ""
}

func ReleaseServer(server collector, addr string) {
	mutex.Lock()
	defer mutex.Unlock()

	if v, ok := v_serverMap[addr]; ok {
		for i,_ := range v {
			if i == server {
				v[i] = true
				break
			}
		}
	}

}

func HandleError(server collector, addr string) {  //错误处理
	status_mutex.Lock()

	if s, ok := v_status[server]; ok {  //  v_status: map[collector]int32
		if s++; s> 3 {
			seelog.Warn("send to server fail more than 3s, remove it from hashtable")
			delete(v_status, server)
			status_mutex.Unlock()
			deleteServer(server, addr)
			closeServer(server)
			addServer(addr)
			return
		} else {
			v_status[server]++
		}
	}

	status_mutex.Unlock()

	return
}

func UpdateServerStatus (server collector) {
	status_mutex.Lock()
	defer status_mutex.Unlock()

	if _, ok := v_status[server]; ok {
		v_status[server] = 0
	}

	return
}

func GetServerInfo() []string {
	var ss []string

	mutex.Lock()
	defer mutex.Unlock()
	for v,_ := range v_serverMap {
		ss = append(ss, v)
	}

	return ss
}