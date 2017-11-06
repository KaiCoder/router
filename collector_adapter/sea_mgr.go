package collector_adapter

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/cihub/seelog"
	net_int "../gen-go/net_interface"
	"time"
)

type Sea_mgr struct {
	transportFactory *thrift.TBufferedTransportFactory
	protocolFactory  *thrift.TBinaryProtocolFactory
	transport *thrift.TSocket
	useTransport     *thrift.TTransport
	server *net_int.CountServiceClient
	addr string
}

func (s *Sea_mgr) Init(addr string) (*net_int.CountServiceClient, error){  //建立连接
	s.addr = addr;
	s.transportFactory = thrift.NewTBufferedTransportFactory(10240)
	s.protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(addr, time.Duration(TIMEOUT)*time.Second)
	s.transport = transport

	if err != nil{
		seelog.Errorf("new socket addr<%s> error:%v",addr,err)
		return nil, err
	}else{
		seelog.Infof("connect to server %s success",addr)
	}

	s.transport.SetTimeout(time.Duration(TIMEOUT)*time.Second)

	useTransport,err := s.transportFactory.GetTransport(s.transport)
	s.server = net_int.NewCountServiceClientFactory(useTransport, s.protocolFactory)
	if err := s.transport.Open();err!=nil{
		seelog.Errorf("socket open addr<%s> error:%v", addr, err)
		return nil, err
	}
	return s.server, err
}

func (s *Sea_mgr) Fini() (err error){ //关闭连接
	err = s.transport.Close()
	if err != nil{
		seelog.Error("connect released err is : %v",err)
		return err
	}
	return err
}

func (s *Sea_mgr) Reset() (err error){  //重连
	seelog.Debug("reset enter")
	err = s.Fini()  //关闭连接
	if err != nil{
		seelog.Error("reset rmq fini err is : %v",err)
		return err
	}
	s.server, err = s.Init(s.addr) //建立连接
	if err != nil{
		seelog.Error("reset rmq init err is : %v",err)
		return err
	}
	return err
}











