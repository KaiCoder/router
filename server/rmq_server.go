package server

import (
	"../gen-go/rmq_service"
	"../collector_adapter"
	cfg "../cfg_adapter"

	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/cihub/seelog"

	"time"
	"strings"
	"fmt"
	"bufio"
	"os"
	"strconv"
)

var isTerminate bool

func init() {
	isTerminate = false
}

func helper() {
	scan := bufio.NewScanner(os.Stdin)

	for scan.Scan() {
		in := strings.ToLower(scan.Text())
		if strings.Compare(in, "help") == 0 {
			fmt.Println(" help: help\r\n server: server information\r\n q: exit\r\n")
		} else if strings.Compare(in, "server") == 0 {
			collector_adapter.PrintServerInfo()
		} else if strings.Compare(in, "q") == 0 {
			isTerminate = true
			break
		} else {
			fmt.Println("No command found")
		}
	}
}

func newTopicServer() ([]*rmq_service.MTRMessageServiceClient) {
	mqadds := strings.Split(cfg.MQSERVER["addr"], ",")
	length := len(mqadds)
	server_list := make([]*rmq_service.MTRMessageServiceClient, 0)

	for {
		for i:=0;i<length;i++ {
			transportFactory := thrift.NewTBufferedTransportFactory(10240)
			protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

			transport,err := thrift.NewTSocketTimeout(mqadds[i], 10*time.Second)
			//transport,err := thrift.NewTSocket(mqadds[i])
			if err != nil {
				seelog.Errorf("connectToServer addr[%s]: NewTSocket error:", mqadds[i], err)
				continue
			}

			transport.SetTimeout(time.Duration(collector_adapter.TIMEOUT)*time.Second)

			useTransport, err := transportFactory.GetTransport(transport)
			server := rmq_service.NewMTRMessageServiceClientFactory(useTransport, protocolFactory)

			if err := transport.Open(); err != nil {
				seelog.Errorf("connectToServer addr[%s]: open error:", mqadds[i], err)
				continue
			} else {
				//return server
				server_list = append(server_list , server)
			}
		}
		return server_list

		//10s秒后再试
		time.Sleep(10*time.Second)
	}

	return nil
}

func StartServer() {
	seelog.Debug("server is started...")

	//get thread size
	thread_num := 10
	if v, err := strconv.Atoi(cfg.MQSERVER["thread_num"]); err == nil {
		thread_num = v
	}

	//get group and topics
	group := cfg.MQSERVER["group"]
	calc_topics := cfg.MQSERVER["calc_topics"]
	sea_topics := cfg.MQSERVER["sea_topics"]
	topics := make(map[string]int32)

	for _,v := range strings.Split(calc_topics, ",") {
		topics[v] = 0 //非汇总
	}
	for _,v := range strings.Split(sea_topics, ",") {
		topics[v] = 1 //汇总
	}

	mapsize := len(topics)
	done := make(chan struct{}, mapsize)

	fmt.Printf("%v %d\n", topics, mapsize)

	//start go thread by topics
	range_num := 1
	for k,v := range topics {
		if v == 0 {
			range_num = thread_num
		} else {
			range_num = 1
		}

		fmt.Printf("range_num =%d \n", range_num)

		for i := 0; i < range_num; i++ {
			go func(m_topic string, m_type int32) {
				//create new server by topic
				server_list_ := newTopicServer()

				for !isTerminate {
					for _, server := range server_list_{
						msg, err := server.Consume(nil, m_topic, group)
						if err != nil {
							if oe, ok := err.(*rmq_service.MTRRPCException); ok {
								switch oe.ID {
								case rmq_service.MTRRPCErrorCode_MTR_SUCCESS:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_RPC_ERROR_NO_MORE_DATA:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_RPC_MESSAGE_IS_NULL:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_TOPIC_NOT_EXITST:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_GROUP_NOT_EXITST:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_CONSUME_NO_MORE_DATA:
									seelog.Infof("receive message from rmq topic<%s>: %s", m_topic, oe)
									time.Sleep(100*time.Millisecond)
									break
								case rmq_service.MTRRPCErrorCode_MTR_RPC_ERROR_BASE:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_ERROR_BASE:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_UNKNOW_HOST_EXCEPTION:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_RPT_CONSUME_START_EXCEPTION:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_RPT_CONSUME_SUBSCRIBE_EXCEPTION:fallthrough
								case rmq_service.MTRRPCErrorCode_MTR_CONSUME_START_EXCEPTION:
									//reconnect
									seelog.Errorf("receive message from rmq topic<%s>: %s", m_topic, oe)
									server_list_ = newTopicServer()
									break
								default:
									seelog.Warnf("default receive topic<%s>: %s", m_topic, oe)
									break
								}
							} else if oe, ok := err.(thrift.TApplicationException); ok {
								seelog.Errorf("TApplicationException receive topic<%s>, TApplicationException %v", m_topic, oe)
							} else {
								seelog.Errorf("exception receive message from rmq topic<%s>: %v", m_topic, err)
								server_list_ = newTopicServer()
							}
						}else {
							if msg != nil && msg.Body != nil{
								seelog.Infof("receive message from rmq topic<%s>: %s", m_topic, msg.Body)
								if collector_adapter.HandleMessage(msg.Body, m_type) != nil {
									//暂时不对失败数据做处理

								}
							} else {
								seelog.Warnf("receive message from rmq topic<%s>:message is null", m_topic)
							}
						}
					}
				}

				done <- struct{}{}
			}(k, v)
		}
	}

	helper()

	for i:=0; i<mapsize; i++ {
		<- done
	}
}