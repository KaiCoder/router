package server

import (
	"../collector_adapter"
	cfg "../cfg_adapter"

	"github.com/cihub/seelog"

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

//server_list := make([]*rmq_service.MTRMessageServiceClient, 0)

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

	mqadds := strings.Split(cfg.MQSERVER["addr"], ",")
//	length := len(mqadds)

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
	for k,v := range topics {  //k: topic  v:汇总or非汇总
		if v == 0 {
			range_num = thread_num  //非汇总 0
		} else {
			range_num = 1 //汇总 1
		}

		fmt.Printf("range_num =%d \n", range_num)

		for i := 0; i < range_num; i++ {
			go func(m_topic string, m_type int32) {
				var rmqInst RmqConn
				rmqInst.RmqInit(mqadds)
				for !isTerminate {
					msg, err := rmqInst.receive(m_topic,group)	//随机选取可用连接，进行消费，出错将连接加入failedlist，成功则返回msg
						if err != nil {
							seelog.Error("Init is failed: %v",err)
							//go rmqInst.reconnect() //检查错误列表，重连 单独开启一个协程处理重连
							//go rmqInst.healthCheck()
							continue
							//
						}else{
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
				done <- struct{}{}  //传一个结构体到done
			}(k, v)
		}
	}
	helper()

	for i:=0; i<mapsize; i++ {   //从done中取，如果为空，则阻塞master，否则正常退出
		<- done
	}
	//防止主线程退出
}
