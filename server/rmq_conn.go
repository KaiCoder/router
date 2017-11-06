package server

import (
	"errors"
	"math/rand"
	"time"
	"../gen-go/rmq_service"
	"github.com/cihub/seelog"
)

type RmqConn struct {
	RmqInstNormal []Rmq_mgr   //有效连接
	RmqInstFailed []Rmq_mgr   //无效连接
}

func (r *RmqConn) RmqInit(addrs []string) (err error){
	err = r.connect(addrs) //连接rmq，可以为多个
	if err != nil && len(r.RmqInstNormal) == 0{   //一个可用的连接都没有，直接退出
		return err
	}
	go r.healthCheck()  //定时进行健康检查，重连无效的连接
//	go r.consumeMsg(topic,group)
	return
}

func (r *RmqConn) connect(addrs []string) (err error){  //管理连接，无效的连接放入无效队列中
	for _,addr := range addrs{
		var rmqMgr Rmq_mgr
		err = rmqMgr.Init(addr)
		if err != nil{
			r.RmqInstFailed = append(r.RmqInstFailed,rmqMgr) //
			seelog.Errorf("addr:%s connect rmq failed",addr)
			continue
		}
		r.RmqInstNormal = append(r.RmqInstNormal,rmqMgr)
	}
	if len(r.RmqInstFailed) == len(addrs){
		err = errors.New("all host is failed")
		return err
	}
	return
}

//func (r *RmqConn) consumeMsg(topic string, group string){
//	defer func() {
//		if errStr := recover();errStr != nil{
//			seelog.Errorf("consumeMsg occur panic")
//		}
//	}()

//	for{
//		select {
//		case msg, ok := <-msgChan:
//			if ok{
//				r.receive(topic, group)
//			}
//		}
//	}
//}

func (r *RmqConn) receive(topic string, group string) (msg *rmq_service.MTRMessage, err error){
	defer func() {
		if errStr := recover(); errStr != nil{
			seelog.Errorf("reconnect occur panic")
		}
	}()

	if len(r.RmqInstNormal) > 0{
		seelog.Debugf("recv msg, RmqInstNormal len is:%d",len(r.RmqInstNormal))
		var nomalInstLen = len(r.RmqInstNormal)

		seed := rand.New(rand.NewSource(time.Now().UnixNano()))
		var index = seed.Intn(nomalInstLen)  //随机数在[0,index)

		msg, err := r.RmqInstNormal[index].ConsumeMsg(topic,group)
		if err != nil{
			//增加到连接异常切片中
			seelog.Debugf("recv msg fail, will append to RmqInstFailed")
			r.RmqInstFailed = append(r.RmqInstFailed,r.RmqInstNormal[index])
			//从连接正常的切片中删除
			var rmqInstTmp []Rmq_mgr
			for i, rmqInst := range r.RmqInstNormal{
				if index != i {
					rmqInstTmp = append(rmqInstTmp,rmqInst)
				}
			}
			r.RmqInstNormal = rmqInstTmp
			seelog.Debugf("RmqInstNormal len is:%d, RmqInstFailed len is:%d",len(r.RmqInstNormal),len(r.RmqInstFailed))
		}else{
			return msg, err
		}
	}
	return msg, err
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
			seelog.Errorf("rmq:%s reset error:%s",rmqInst.addr,err.Error())
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
		seelog.Debugf("rmq %v reconnect is success!!!",rmqInst.addr)
	}
	return
}

//若问题列表中有连接，则尝试重连
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
