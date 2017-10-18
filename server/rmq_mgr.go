package server

import (
	"../gen-go/rmq_service"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/cihub/seelog"

	"time"
	"errors"
	"strings"
	"fmt"
	"bufio"
	"os"
	"strconv"
)

type Rmq_mgr struct {
	transportFactory *thrift.TBufferedTransportFactor
	protocolFactory  *thrift.TBinaryProtocolFactory
	transport        *thrift.TSocket
	useTransport     *thrift.TTransport
	server           *rmq_adapter.MTRMessageServiceClient
	host             string
	port             string
}

func (r.Rmq_mgr) Init(host string, port string) (err error){
	r.host = host;
	r.port = port;

	addr := fmt.Sprintf("%v:%v",host,port)

	r.transportFactory = thrift.NewTBufferedTransportFactory(10240)
	r.protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	r.transport,err = thrift.NewTSocketTimeout(addr, 10*time.Second)
	if err != nil{
		seelog.Error("thrift.NewTSocket err is: %v",err);
		return err;
	}else{
		seelog.Debugf("connected %v successful",addr);
	}

	useTransport := r.transportFactory.GetTransport(r.transport)
	r.server = rmq_service.NewMTRMessageServiceClientFactory(useTransport, protocolFactory)

	if err := r.transport.Open();err != nil{
		seelog.Errorf("connectToServer : open error:", err)
		return err
	}
	return err
}

func (r *Rmq_mgr) Fini() (err error){
	err = r.transport.Close()
	if err != nil{
		seelog.Error("connect released err is : %v",err);
	}
	return
}

//重连rmq
func (r *Rmq_mgr) Reset() (err error){
	seelog.Debug("reset enter")
	err = e.Fini()
	if err != nil{
		seelog.Error("reset rmq fini err is : %v",err)
		return
	}
	err = r.Init(r.host,r.port)
	if err != nil{
		seelog.Error("reset rmq init err is : %v",err)
		return err
	}
	return
}

func (r *Rmq_mgr) ConsumeMsg(topic string,body string) (consume_reply int64, consume_err error){
	var msg rmq_adapter.MTRMessage
	msg.Topic = topic
	msg.Body = []byte(body) //强转成字节类型的切片
	msg.Protocol = rmq_adapter.MTRProtocol_PERSONALIZED

	var resendMaxTimes, resendTime =3, 0
	for{
		if resendTime > resendMaxTimes{
			consume_err = errors.New("resend three times failed")
			seelog.Error("resend three times failed")
			return consume_reply,consume_err
		}
		consume_reply, consume_err = r.server.Consume(&msg,true)
		if consume_err != nil{
			resendTime += 1
			continue
		}
		seelog.Debugf("send msg success")
		return
	}
	return
}