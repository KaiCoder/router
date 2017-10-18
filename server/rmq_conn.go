package server

import (
	"errors"
	"math/rand"
	"time"

	"github.com/cihub/seelog"
)

type RmqConn struct {
	RmqInstNormal []Rmq_mgr
	RmqInstFailed []Rmq_mgr
}

func (r *RmqConn) RmqInit(topic string, hosts []string, port string, msgChan chan string) (err error){
	err = r.connect(hosts,port)
	if err != nil && len(r.RmqInstNormal) == 0{   //一个可用的连接都没有，直接退出
		return err
	}

	go r.healthCheck()
	go r.consumeMsg(topic,msgChan)
	return
}

func (r *RmqConn) connect(hosts []string, port string) (err error){
	for _,host := range hosts{
		var rmqMgr Rmq_mgr
		err = rmqMgr.Init(host,port)
		if err != nil{
			r.RmqInstFailed = append(r.RmqInstFailed,rmqMgr)
			seelog.Errorf("host:%s connect rmq failed",host)
			continue
		}
		r.RmqInstNormal = append(r.RmqInstNormal,rmqMgr)
	}
	if len(r.RmqInstFailed) == len(hosts){
		err = errors.New("all host is failed")
		return err
	}
	return
}

func (r *RmqConn) reconnect() (err error){
	defer func(){
		if errStr := recover();errStr != nil{
			seelog.Errorf("reconnect occur panic")
		}
	}()

	for i, rmqInst := range r.RmqInstFailed{
		err = rmqInst.Reset()
		if err != nil{
			seelog.Errorf("rmq:%s reset error:%s",rmqInst.host,err.Error())
			continue
		}

		r.RmqInstNormal = append(r.RmqInstNormal,r.RmqInstFailed[i])
		var rmqInstTmp []Rmq_mgr
		for index, rmqInst := range r.RmqInstFailed{
			if index != i{
				rmqInstTmp = append(rmqInstTmp,rmqInst)
			}
		}
		r.RmqInstFailed = rmqInstTmp;
		seelog.Debugf("rmq %v reconnect is success!!!",rmqInst.host)
	}
	return
}

func (r *RmqConn) healthCheck(){
	for{
		seelog.Debug("healthcheck:")
		time.Sleep(1 * time.Second)
		seelog.Debugf("RmqInstFailed len:%d",len(r.RmqInstFailed))
		if len(r.RmqInstFailed) >= 0{
			r.reconnect()
		}
	}
}
func (r *RmqConn) Fini(){
	for _, inst := range r.RmqInstFailed{
		inst.Fini()
	}
	for _, inst := range r.RmqInstNormal{
		inst.Fini()
	}
}

