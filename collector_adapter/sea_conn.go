package collector_adapter

import (
	"time"
	cfg "../cfg_adapter"
	xconf "go-xConf"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/cihub/seelog"
	"sync"
	"encoding/json"
	"strconv"
)

//consistent hash
var hash *HashRing

//connect poolsize
var poolsize int
var TIMEOUT int

type server_status map[Sea_mgr]bool
//pool 状态
var v_server map[string]server_status
var v_serverMap map[string]bool  //仅存储对应的sea对应的addr

var mutex sync.Mutex

//map key:server-addr value:collector sets,connect pool
//var v_serverMap map[string]collector_set

type handle struct {
}

//非汇总
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

//map[string]Sea_mgr   addr:sea服务地址  Sea_mgr:具体链接
type SeaConn struct {
	SeaInstNormal []map[Sea_mgr]bool  //正常可用的连接
	SeaInstFailed []map[Sea_mgr]bool  //有问题的连接
}
var s *SeaConn  //

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
			err := SeaInit(addr)  //默认每个sea服务建立10个连接
			if err == nil {
				hash.AddNode(addr, 1)
			}
		}
		set[addr] = true
	}

	//remove offline server
	for addr,_ := range v_serverMap {
		if _,ok := set[addr]; !ok {
			hash.RemoveNode(addr)
			delete(v_serverMap, addr) //在v_serverMap删除sea服务的addr
		}
	}
	return err
}

func init(){
	virtualSpots := 100
	hash = NewHashRing(virtualSpots)
	if v,err := strconv.Atoi(cfg.CONNTECTPOOL["poolsize"]);err == nil{
		poolsize = v
	}else {
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

			//写入列表
			for _, addr := range result{   //根据不同的sea addr 创建多个连接 addr个数*10
				if _, ok := v_serverMap[addr]; !ok{   //v_serverMap : 存储sea服务对应的addr
					err := SeaInit(addr)  //每个addr创建10个连接
					if err == nil{
						//server_status
						//v_serverMap[addr] = true     //
						hash.AddNode(addr,1)  //
					}
				}
			}
		}
		for{
			time.Sleep(5*time.Second)
		}
		s.Fini()
	}()
}

func SeaInit(addr string) (err error){  //初始化连接
	seelog.Infof("poolsize=%d",poolsize)
	v_serverMap[addr] = true

	for i:=0;i< poolsize;i++{
		var seaMgr Sea_mgr
		_,err = seaMgr.Init(addr)
		if err != nil{
			var m = make(map[Sea_mgr]bool)
			m[seaMgr] = false  //表示不可用，可能是失败的连接 或者已经占用的连接
			s.SeaInstFailed = append(s.SeaInstFailed,m)  //加入无效的连接队列中
			//s.SeaInstFailed = append(s.SeaInstFailed,seaMgr)  //连接失败，放入failed队列
			//v_server[addr][seaMgr] = false
			seelog.Errorf("addr:%s connect sea failed",addr)
		}else{
			//s.SeaInstNormal = append(s.SeaInstNormal,seaMgr)  //连接成功，放入normal队列
			var m = make(map[Sea_mgr]bool)
			m[seaMgr] = true
			s.SeaInstNormal = append(s.SeaInstNormal,m)
			//v_server[addr][seaMgr] = true
			seelog.Infof("addr:%s connect sea success",addr)
		}
	}
	return err
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

func GetServerByAppID(appID string)(sea_mgr Sea_mgr,status bool) {
	mutex.Lock()
	defer mutex.Unlock()

	addr := hash.GetNode(appID)   //根据appID获取节点
	if _, ok := v_serverMap[addr]; ok {
		if len(s.SeaInstNormal) > 0 {
			seelog.Debugf("SeaInstNormal len is : %d", len(s.SeaInstNormal))
			for _ , j := range s.SeaInstNormal {
				for sea, st := range j{
					if st{  // status = true 表示连接可用
						return sea,st
					}
				}
			}
		}
	}
	var s Sea_mgr //
	return s,false
}

func reconnect() (err error){
	for i,seaInst := range s.SeaInstFailed{
		for sea ,_ := range seaInst{
			err := sea.Reset() //重连
			if err != nil{
				seelog.Errorf("sea:%s reset error:%s",sea.addr,err.Error())
				continue
			}
			s.SeaInstNormal = append(s.SeaInstNormal,s.SeaInstFailed[i])
			//var seaInstTmp []
			var seaInstTmp = make([]map[Sea_mgr]bool,20)
			//var m = make(map[Sea_mgr]bool)
			for index, seaInst := range s.SeaInstFailed{
				if index != i{
					seaInstTmp = append(seaInstTmp,seaInst)
				}
			}
			s.SeaInstFailed = seaInstTmp
			seelog.Debugf("sea %v reconnect is success!!!",sea.addr)
		}
	}
	return
}


func Fini(){
	for _, inst := range s.SeaInstFailed{
		for seaInst,_ := range inst{
			seaInst.Fini()
		}
	}
	for _, inst := range s.SeaInstNormal{
		for seaInst,_ := range inst{
			seaInst.Fini()
		}
	}
}

func GetServerInfo() []string{
	var ss []string

	mutex.Lock()
	defer mutex.Unlock()

	for v,_ := range v_serverMap{
		ss = append(ss,v)
	}
	return ss
}

func HandleError(sea_mgr Sea_mgr){

}













