package server

import (
	"../gen-go/rmq_service"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/cihub/seelog"

	"time"
	"errors"
)

type Rmq_mgr struct {  //连接结构体
	transportFactory *thrift.TBufferedTransportFactory
	protocolFactory  *thrift.TBinaryProtocolFactory
	transport        *thrift.TSocket
	useTransport     *thrift.TTransport
	server           *rmq_service.MTRMessageServiceClient
	addr             string  //host:port
}

func (r *Rmq_mgr) Init(addr string) (err error){  //建立连接
	r.addr = addr;
	r.transportFactory = thrift.NewTBufferedTransportFactory(10240)
	r.protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	r.transport,err = thrift.NewTSocketTimeout(addr, 10*time.Second)
	if err != nil{
		seelog.Error("thrift.NewTSocket err is: %v",err);
		return err;
	}else{
		seelog.Debugf("connected %v successful",addr);
	}

	useTransport, err := r.transportFactory.GetTransport(r.transport)
	r.server = rmq_service.NewMTRMessageServiceClientFactory(useTransport, r.protocolFactory)

	if err := r.transport.Open();err != nil{
		seelog.Errorf("connectToServer : open error:", err)
		return err
	}
	return err
}

func (r *Rmq_mgr) Fini() (err error){  //关闭连接
	err = r.transport.Close()
	if err != nil{
		seelog.Error("connect released err is : %v",err);
	}
	return
}

//重连rmq
func (r *Rmq_mgr) Reset() (err error){  //重连rmq
	seelog.Debug("reset enter")
	err = r.Fini()   //关闭连接
	if err != nil{
		seelog.Error("reset rmq fini err is : %v",err)
		return err
	}
	err = r.Init(r.addr)  //建立连接
	if err != nil{
		seelog.Error("reset rmq init err is : %v",err)
		return err
	}
	return err
}

//根据Topic和Group消费
func (r *Rmq_mgr) ConsumeMsg(topic string,group string) (msg *rmq_service.MTRMessage, err error){  //根据topic和group进行消费
	var retryMaxTimes, retryTime =3, 0  //重试次数
	for{
		if retryTime > retryMaxTimes{
			err = errors.New("retry three times failed")
			seelog.Error("retry three times failed")
			return msg,err
		}
		msg, err = r.server.Consume(nil,topic,group)  //rmq消费
		if err != nil{
			retryTime += 1
			continue
		}
		seelog.Debugf("consume msg success")
		return msg, err
	}
	return
}
