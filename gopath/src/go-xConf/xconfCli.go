// Configuration Center & Service Discovery SDK
package xconf

import (
	"errors"
	log "github.com/cihub/seelog"
	"github.com/samuel/go-zookeeper/zk"
	"go-xConf/common/zkcli"
	"net"
	"path"
	"time"
)

var exitChan chan bool
var logger log.LoggerInterface

const (
	version = "1.0.0.4"
)

// CfgHandler config file update Handler
type CfgHandler interface {
	HandlerCfgUpdate(cfgMsg map[string][]byte, err error) error
}

// SrvHandler server nodes update handler
type SrvHandler interface {
	//TODO add srv and tags
	HandlerSvrUpdate(srvMsg SrvMessage, err error) error
}

type SrvMessage struct {
	Srv     string
	Tags    []string
	SrvAddr []string
}

// cfg message struct
type configFile struct {
	Name    string `json:"name"`
	Content []byte `json:"content"`
	MD5     string `json:"md5"`
}

type configFileArray struct {
	Ver         int64        `json:"ver"`
	ConfigArray []configFile `json:"configs"`
}

// Service
type Service struct {
	srv             string
	tags            []string
	zkConn          *zk.Conn
	zm              *ZkMonitor
	xMgr            *xManager
	httpHandler     *httpHandler
	dumpfileHandler *dumpHandler
	hp              *xHostProvider

	localAddr string
	StopCH    chan int
}

// NewService using srv,tags and addr to create a service
// srv refer to the name of the service create by yourself.
// tags refer to the service tags and address refer to the
// address will be used in service registration and config
// update feedback.
func NewService(srv string, tags []string, addr string) (*Service, error) {
	var err error
	err = newxConfLogger(srv)
	if err != nil {
		return nil, err
	}

	logger.Trace("xConf NewService")
	defer logger.Trace("xConf NewService Leave")
	if len(addr) == 0 {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return nil, err
		}

		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					addr = ipnet.IP.String()
					addr = addr + ":xxxx" //default port
					break
				}
			}
		}
	}

	if len(srv) == 0 {
		err = errors.New("wo can't do out job without srv,srv is nil")
		return nil, err
	}

	s := &Service{
		srv:             srv,
		tags:            tags,
		localAddr:       addr,
		StopCH:          make(chan int),
		httpHandler:     newHttpHandler(),
		dumpfileHandler: newDumpHandler(srv),
		zm:              NewZkMonitor(),
		xMgr:            newXconfManager(),
		hp:              &xHostProvider{},
	}

	return s, err
}

// Init return err when init failed.The init function
// provided a set of request address to implement service
// polling.if requestAddr is nil, xConf will get the address
// from the DNS(s.xconf.openspeech.cn).
func (s *Service) Init(requestAddr []string) (err error) {
	logger.Trace("xConf Init")
	defer logger.Trace("xConf Init Leave")

	err = s.httpHandler.SetPostUrl(requestAddr)
	if err != nil {
		return err
	}

	zkAddrs, err := s.httpHandler.getZkAddr()
	if err != nil {
		logger.Error("xConf Init | get zk addrs error:", err)
		return err
	}

	s.httpHandler.zkAddrPolling(s.hp, zkAddrs, 10*time.Minute)

	s.zkConn, err = connectToCluster(zkAddrs, s.hp)
	if err != nil {
		return err
	}
	s.zm.setZkConn(s.zkConn)
	zkcli.AddMonitor(s.zm)

	// zookeeper logger redirection
	zooLogger := newZooLogger(s.srv)
	s.zkConn.SetLogger(zooLogger)

	return err
}

// Fini xConf Fini
func (s *Service) Fini() error {
	logger.Trace("xConf Fini")
	defer logger.Trace("xConf Fini Leave")
	var err error
	zkcli.RemoveMonitor(s.zm)
	zkcli.Exit <- true
	s.zkConn.Close()
	s.xMgr.reset()
	return err
}

// Register
func (s *Service) Register() (err error) {
	logger.Trace("xConf Register")
	defer logger.Trace("xConf Register Leave")
	var regPath string
	var deRegPath string
	tempMap, err := s.httpHandler.GetPath(s.srv, refType, s.tags)
	if err != nil {
		logger.Error("xConf Register | xConf get register path error", err)
		return err
	}

	//Because the current GetPath interface is not separated, resulting in the existence of the Service address array
	for srvPath, _ := range tempMap {
		regPath = path.Join(srvPath, s.localAddr)
	}

	deRegPath, err = s.zkConn.Create(regPath, nil, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		logger.Error("xConf Register | xConf zk create node error:", err)
		return err
	}
	// set the path after create path to
	s.xMgr.setRegPath(regPath)
	s.xMgr.setDeregPath(deRegPath)
	s.xMgr.setRegFlag(true)

	return err
}

// Deregister delete path in zookeeper
func (s *Service) Deregister() (err error) {
	logger.Trace("xConf Deregister")
	defer logger.Trace("xConf Deregister Leave")
	err = s.zkConn.Delete(s.xMgr.getDeregPath(), -1)
	if err != nil {
		logger.Error("xConf Deregister | server deregister error:", err)
	}

	return err
}

// GetCfg get configuration
// A return to all configuration file at the beginning,
// followed by a callback notification via the interface
// when there is a change in configuration files.
func (s *Service) GetCfg(cfgHandler CfgHandler) (result map[string][]byte, err error) {
	logger.Trace("xConf GetCfg")
	defer logger.Trace("xConf GetCfg Leave")
	allCfgDataArray := make([]configFile, 0)
	result = make(map[string][]byte)

	getCfgSuccess := true
	if cfgHandler == nil {
		logger.Warn("XConf GetCfg | get config handler is nil")
	}

	cfgMap, err := s.httpHandler.GetPath(s.srv, cfgType, s.tags)
	if err != nil {
		logger.Error("xConf GetCfg | get cfg path err", err)
		result, err = s.dumpfileHandler.readDumpFile()
		return result, err
	}

	for cfgPath, needWatched := range cfgMap {
		logger.Info("xConf GetCfg | cfgPath:", cfgPath, "isWatched:", needWatched)
		var cfgArray []configFile
		cfgArray = make([]configFile, 0)
		var ver int64
		if needWatched == 1 {
			//add cfgUpdateMess to xMgr for disconnection
			cfgArray, ver, err = getDynamicCfg(cfgPath, cfgHandler, s.zkConn, s.srv, s.localAddr)
			s.xMgr.setCfgWatchFlag(true)
			s.xMgr.addCfgUpdateMess(s.srv, s.localAddr, cfgPath, cfgHandler)
			if err != nil {
				logger.Error("xConf GetCfg | get dynamic config error:", err)
				getCfgSuccess = false
				s.httpHandler.updateFeedback(s.srv, cfgPath, ver, s.localAddr, 1, err.Error())
				continue
			}
			go s.dumpfileHandler.writeDynamicDumpFile(cfgArray)
			s.httpHandler.updateFeedback(s.srv, cfgPath, ver, s.localAddr, 0, "update success")
		} else {
			cfgArray, ver, err = getStaticCfg(cfgPath, s.zkConn)
			if err != nil {
				logger.Error("xConf GetCfg | get static config error:", err)
				getCfgSuccess = false
				s.httpHandler.updateFeedback(s.srv, cfgPath, ver, s.localAddr, 1, err.Error())
				continue
			}
			xMgr.addStaticCfg(cfgArray)
			go s.dumpfileHandler.writeStaticDumpFile(cfgArray)

			s.httpHandler.updateFeedback(s.srv, cfgPath, ver, s.localAddr, 0, "update success")
		}

		for _, v := range cfgArray {
			allCfgDataArray = append(allCfgDataArray, v)
		}

	}

	if !getCfgSuccess {
		result, err = s.dumpfileHandler.readDumpFile()
		return result, err
	}

	for _, v := range allCfgDataArray {
		result[v.Name] = v.Content
	}

	return result, err
}

//TODO get Customized configuration
func (s *Service) GetCustomizedCfg(svr string, tags []string, isWatch bool) (err error) {
	return nil
}

// GetSrvNodes returns the srv nodes as string array.
// A return to all registered nodes at the beginning,
// followed by a callback notification via the interface
// when there is a change.
func (s *Service) GetSrvNodes(srv string, tags []string, srvHandler SrvHandler) (result []string, err error) {
	logger.Trace("xConf GetSrvNodes")
	defer logger.Trace("xConf GetSrvNode Leave")
	var srvPath string
	//Because the current GetPath interface is not separated, resulting in the existence of the Service address array
	tempMap, err := s.httpHandler.GetPath(srv, refType, tags)
	if err != nil {
		logger.Error("xConf GetSrvNodes | get path error:", err)
		return result, err
	}

	for path, _ := range tempMap {
		srvPath = path
	}

	if srvHandler != nil {
		logger.Debug("xConf GetSrvNodes | watch srv node path:",srvPath)
		result, err = srvNodesWatch(s.zkConn, srvPath, srvHandler, srv, tags)
		//
		s.xMgr.setSrvWatchFlag(true)
		s.xMgr.addSrvDiscoverMess(srv, tags, srvPath, srvHandler)
	} else {
		err = errors.New("xConf GetSrvNodes | srv discovery handler is nil")
	}

	return result, err
}
