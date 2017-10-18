package server

import (
	"../gen-go/rmq_service"

	"git.apache.org/thrift.git/lib/go/thrift"
	log "github.com/cihub/seelog"

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
	r.protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	r.transport,err := thrift.NewTSocketTimeout(mqadds[i], 10*time.Second)
	if err != nil{
		log.Error("thrift.NewTSocket err is: %v",err);
		return err;
	}else{
		log.Debugf("connected %v successful",addr);
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

func (r *Rmq_mgr) Reset() (err error){
	seelog.Debug("reset enter")
	err = e.Fini()



}