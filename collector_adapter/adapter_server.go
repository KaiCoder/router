package collector_adapter

import (
	"github.com/cihub/seelog"
	"errors"
	"fmt"
	"strings"
)

func init() {

}

func SendMessage(msg []byte, appid string) error { //向sea发送消息
	srv,status := GetServerByAppID(appid) //根据AppID获取sea服务  返回*net_int.CountServiceClient和string
	if !status{
	//	if strings.Compare(addr , "flag") == 0 {
	//		seelog.Warn("not found server in current pool")
	//	} else {
			seelog.Warnf("not found server by appID<%s>", appid)
	//	}
		return errors.New("not found server by appID")
	}

	//send message
	intval, err := srv.server.SendMessageToServer(nil, string(msg[:]))
	//ReleaseServer(server, addr)
	if err != nil {
		seelog.Errorf("send message to collector_server fail %s", err)
		HandleError(srv)
		return err
	} else {
		seelog.Infof("send message to collector_server success,retval <%d>", intval)
		UpdateServerStatus(srv)
		return nil
	}
}

func HandleMessage(msg []byte, m_type int32) error { //处理rmq消息
	//parse appid uid, get server by hash    解析appid，uid  根据一致性哈希算法获取sea服务
	switch m_type {
	case 0:  //非汇总
		if appid, err := ParseMessage(msg); err == nil {
			return SendMessage(msg, appid)
		} else {
			seelog.Error("json message parse error")
			return errors.New("json message parse error")
		}
	case 1:  //汇总
		if appids, err := ParseMultiMessage(msg); err == nil {
			for appid, msg := range appids {
				SendMessage(msg, appid)
			}
		} else {
			seelog.Error("json multi message parse error")
			return errors.New("json multi message parse error")
		}
	default:
		seelog.Error("not found parse type")
		return errors.New("not found parse type")
	}
	return nil
}

func PrintServerInfo() {
	server := GetServerInfo()
	for _,k := range server {
		fmt.Printf("[server info:%s]", k)
	}
	fmt.Println("")
}