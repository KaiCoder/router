package xconf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"
)

const (
	cfgType         = 1
	refType         = 2
	pathRequest     = 3
	feedbackRequest = 4
	zkAddrRequest   = 5
)

const (
	xDNS  = "s.xconf.openspeech.cn"
	xPort = ":2405"
)

const (
	pathSuffix     = "/xconf/v1/path/get.do"
	feedbackSuffix = "/xconf/v1/update/feedback.do"
	zkAddrSuffix   = "/xconf/v1/zkaddrs/query.do"
	httpPrefix     = "http://"
)

var httphdl *httpHandler
var httpOnce sync.Once

//path request struct
type pathRequestData struct {
	Srv        string   `json:"srv"`
	PathType   int      `json:"pathType"`
	Tags       []string `json:"tags"`
	NeedCreate int      `json:"needCreate"`
}

// path response struct
type pathResponseData struct {
	Ret       int           `json:"ret"`
	PathArray []pathContent `json:"pathArray"`
}

type updateFeedBackData struct {
	Srv        string `json:"srv"`
	Path       string `json:"path"`
	Ver        int64  `json:"ver"`
	IP         string `json:"ip"`
	State      int    `json:"state"`
	Info       string `json:"info"`
	UpdateTime int64  `json:"updateTime"`
}

type feedbackResult struct {
	Ret int `json:"ret"`
}

//cfg content struct
type pathContent struct {
	CfgPath   string `json:"path"`
	NeedWatch int    `json:"needWatch"`
}

type zkResult struct {
	ZkAddrs    []string `json:"addrs"`
	UpdateTime int64    `json:"updatetime"`
}

type httpHandler struct {
	feedbackPostUrl string
	pathPostUrl     string
	postUrlArray    []string
	postIndex       int
	retryTime       int
	locker          *sync.Mutex
}

func newHttpHandler() *httpHandler {
	httpOnce.Do(func() {
		httphdl = &httpHandler{
			postUrlArray: make([]string, 0),
			locker:       new(sync.Mutex),
			retryTime:    5,
		}
	})

	return httphdl
}

func (h *httpHandler) SetPostUrl(addrs []string) (err error) {
	logger.Trace("xConf SetPostUrl")
	defer logger.Trace("xConf SetPostUrl Leave")

	if len(addrs) == 0 {
		addrs, err = net.LookupHost(xDNS)
		logger.Debug("xConf SetPostUrl | use zk dns to find xconf server,result:", addrs)
		if err != nil {
			logger.Error("xConf SetPostUrl | Look up DNS(xconf.openspeech.cn) error:", err)
			return err
		}
		for k, v := range addrs {
			addrs[k] = v + xPort
		}
	}

	if len(addrs) < 1 {
		return errors.New("xConf SetPostUrl | xconf server addsess is nil")
	}

	h.postUrlArray = addrs

	return err
}

// get configuration file path
func (h *httpHandler) GetPath(srv string, pathType int, tags []string) (pathMap map[string]int, err error) {
	logger.Trace("xConf GetPath")
	postData := pathRequestData{
		Srv:        srv,
		PathType:   pathType,
		Tags:       tags,
		NeedCreate: 0, //whether need to be created
	}

	switch pathType {
	case cfgType:
		postData.NeedCreate = 0
	case refType:
		postData.NeedCreate = 1
	}

	postByte, err := json.Marshal(postData)
	if err != nil {
		logger.Error("xConf GetPath | post struct json marshal error:", err)
		return nil, err
	}
	logger.Debug("json:", string(postByte))

	pathMap = make(map[string]int)

	resultByte, err := h.httpRequest(postByte, pathRequest)
	if err != nil {
		logger.Error("xConf GetPath | http request error:", err)
		return nil, err
	}

	result := pathResponseData{}
	err = json.Unmarshal(resultByte, &result)
	if err != nil {
		logger.Error("xConf GetPath | unmarshal result json error:", err)
		return nil, err
	}

	logger.Debug("un marshal:", result)

	if result.Ret != 0 {
		logger.Error("xConf GetPath | get path result ret:", result.Ret)
		err = errors.New(fmt.Sprintf("ret:%d", result.Ret))
		return nil, err
	}

	for _, v := range result.PathArray {
		pathMap[v.CfgPath] = v.NeedWatch
	}

	return pathMap, err
}

// update feedback
func (h *httpHandler) updateFeedback(srv string, path string, ver int64, ip string, state int, info string) (err error) {
	logger.Trace("xConf updateFeedback")
	defer logger.Trace("xConf updateFeedback Leave")

	feedbackData := updateFeedBackData{
		Srv:   srv,
		Path:  path,
		Ver:   ver,
		IP:    ip,
		State: state,
		Info:  info,
	}
	feedbackData.UpdateTime = time.Now().Unix()

	feedbackByte, err := json.Marshal(feedbackData)
	if err != nil {
		logger.Error("xConf updateFeedback | feedback data marshal error:", err)
	}
	logger.Debug("xConf updateFeedback | feedback post data:", string(feedbackByte))

	resultByte, err := h.httpRequest(feedbackByte, feedbackRequest)
	if err != nil {
		logger.Error("xConf updateFeedback | http request error:", err)
		return err
	}

	result := feedbackResult{}
	err = json.Unmarshal(resultByte, &result)
	if err != nil {
		logger.Error("xConf updateFeedback | json unmarshal error:", err)
		return err
	}

	if result.Ret == 0 {
		logger.Debug("xConf updateFeedback | feedback success,ret:", result.Ret)
	} else {
		logger.Error("xConf updateFeedback | feedback failed,ret:", result.Ret)
	}

	return err
}

// getZkAddr
func (h *httpHandler) getZkAddr() (addr []string, err error) {
	logger.Trace("xConf getZkAddr")
	defer logger.Trace("xConf getZkAddr Leave")

	resultByte, err := h.httpRequest(nil, zkAddrRequest)
	if err != nil {
		logger.Error("xConf updateFeedback | http request error:", err)
		return nil, err
	}

	result := zkResult{}
	err = json.Unmarshal(resultByte, &result)
	if err != nil {
		logger.Error("xConf getZkAddr | json unmarshal error:", err)
		return nil, err
	}
	logger.Debug("xConf getZkAddr | zk addrs:", result.ZkAddrs)

	return result.ZkAddrs, err
}

// httpRequest Handle http request,include path,feedback,zkAddrs
func (h *httpHandler) httpRequest(requestBody []byte, requestType int) (result []byte, err error) {
	logger.Trace("xConf httpRequest")
	defer logger.Trace("xConf httpRequest Leave")

	if len(h.postUrlArray) < 1 {
		return nil, errors.New("xConf httpRequest | post url is nil")
	}

	var req *http.Request
	var resp *http.Response

	body := bytes.NewBuffer(requestBody)
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	for i := 0; i < h.retryTime; i++ {
		for ; h.postIndex < len(h.postUrlArray); h.postIndex++ {
			v := h.postUrlArray[h.postIndex]
			switch requestType {
			case pathRequest:
				req, _ = http.NewRequest("POST", httpPrefix+v+pathSuffix, body)
			case feedbackRequest:
				req, _ = http.NewRequest("POST", httpPrefix+v+feedbackSuffix, body)
			case zkAddrRequest:
				req, _ = http.NewRequest("GET", httpPrefix+v+zkAddrSuffix, body)
			}
			logger.Debug("xConf httpRequest | post url adddress:", v)

			req.Header.Set("Content-Type", "application/json;charset=utf-8")
			resp, err = client.Do(req)
			if err != nil {
				logger.Warn("xConf httpRequest | http request error,try others url address,error:", err)
				continue
			}
			break
		}
		if err == nil {
			break
		}
		logger.Debug("xConf httpRequest | http request err:",err,"retry time:",i)
	}

	if err != nil {
		if h.postIndex == len(h.postUrlArray) {
			h.postIndex = 0
		}
		return nil, err
	}
	defer resp.Body.Close()

	resultByte, err := ioutil.ReadAll(resp.Body)
	logger.Debug("get path body:", string(resultByte))
	if err != nil {
		logger.Error("xConf httpRequest | read result body,err", err)
		return nil, err
	}

	return resultByte, err
}

// xServerPolling
func (h *httpHandler) xServerPolling(duration time.Duration) {
	go func() {
		postUrlCheck := h.postUrlArray
		for {
			addrs, err := net.LookupHost(xDNS)
			logger.Debug("xConf xServerPolling | dns polling,dns result:", addrs)
			if err != nil {
				logger.Error("xConf xServerPolling | Look up DNS(s.xconf.openspeech.cn) error:", err)
				time.Sleep(duration)
				continue
			}
			for k, v := range addrs {
				addrs[k] = v + xPort
			}

			if reflect.DeepEqual(addrs, postUrlCheck) || len(addrs) == 0 {
				logger.Debug("xConf xServerPolling | xserver addrs:", addrs, "addrs check:", postUrlCheck)
				time.Sleep(duration)
				continue
			}

			//success
			h.locker.Lock()
			h.postUrlArray = addrs
			h.locker.Unlock()
			postUrlCheck = addrs
			time.Sleep(duration)
		}
	}()

}

// zkAddrPolling
func (h *httpHandler) zkAddrPolling(provider zk.HostProvider, addrs []string, duration time.Duration) {
	go func() {
		addrsCheck := addrs
		for {
			zkAddr, err := h.getZkAddr()
			if err != nil {
				logger.Warn("xConf zkAddrPolling | get zk address error:", err)
				time.Sleep(duration)
				continue
			}

			if reflect.DeepEqual(zkAddr, addrsCheck) || len(zkAddr) == 0 {
				logger.Debug("xConf zkAddrPolling | compare zk address,zkAddr:", zkAddr, "zkAddrCheck:", addrsCheck)
				time.Sleep(duration)
				continue
			}

			addrsCheck = zkAddr
			err = provider.Init(addrsCheck)
			if err != nil {
				logger.Warn("xConf zkAddrPolling | host provider init error:", err)
			}
			time.Sleep(duration)
		}
	}()
}
