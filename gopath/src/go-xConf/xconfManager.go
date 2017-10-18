package xconf

import (
	"reflect"
	"sync"
)

type xManager struct {
	srvDiscMessArray   []srvDiscoverMess
	cfgUpdateMessArray []cfgUpdateMess
	regPath            string
	deRegPath          string

	addrCache       []string //cache
	dynamicCfgCache map[string][]byte
	staticCfgs      []configFile

	needReg      bool
	needSrvWatch bool
	needCfgWatch bool
}

type srvDiscoverMess struct {
	srv        string
	tags       []string
	srvPath    string
	srvhandler SrvHandler
}

type cfgUpdateMess struct {
	srv       string
	localAddr string
	cfgPath   string
	cfgH      CfgHandler
}

var xOnce sync.Once
var xMgr *xManager

func newXconfManager() *xManager {
	xOnce.Do(func() {
		xMgr = &xManager{
			srvDiscMessArray:   make([]srvDiscoverMess, 0),
			cfgUpdateMessArray: make([]cfgUpdateMess, 0),
			staticCfgs:         make([]configFile, 0),
		}
	})

	return xMgr
}

func (xMgr *xManager) reset() {
	xMgr.needReg = false
	xMgr.needSrvWatch = false
	xMgr.needCfgWatch = false

	xMgr.srvDiscMessArray = xMgr.srvDiscMessArray[:0]
	xMgr.cfgUpdateMessArray = xMgr.cfgUpdateMessArray[:0]

}

func (xMgr *xManager) addSrvDiscoverMess(srv string, tags []string, srvPath string, h SrvHandler) {
	srvDisc := srvDiscoverMess{
		srv:        srv,
		tags:       tags,
		srvPath:    srvPath,
		srvhandler: h,
	}
	xMgr.srvDiscMessArray = append(xMgr.srvDiscMessArray, srvDisc)
}

func (xMgr *xManager) getSrvDiscoverMessArray() []srvDiscoverMess {
	return xMgr.srvDiscMessArray
}

func (xMgr *xManager) addCfgUpdateMess(srv string, localAddr string, cfgPath string, h CfgHandler) {
	cfgMess := cfgUpdateMess{
		srv:       srv,
		localAddr: localAddr,
		cfgPath:   cfgPath,
		cfgH:      h,
	}
	xMgr.cfgUpdateMessArray = append(xMgr.cfgUpdateMessArray, cfgMess)
}

func (xMgr *xManager) getCfgUpdateMessArray() []cfgUpdateMess {
	return xMgr.cfgUpdateMessArray
}

func (xMgr *xManager) getRegPath() string {
	return xMgr.regPath
}

func (xMgr *xManager) setRegPath(path string) {
	xMgr.regPath = path
}

func (xMgr *xManager) getDeregPath() string {
	return xMgr.deRegPath
}

func (xMgr *xManager) setDeregPath(path string) {
	xMgr.deRegPath = path
}

func (xMgr *xManager) setRegFlag(flag bool) {
	xMgr.needReg = flag
}

func (xMgr *xManager) getRegFlag() bool {
	return xMgr.needReg
}

func (xMgr *xManager) setSrvWatchFlag(flag bool) {
	xMgr.needSrvWatch = flag
}

func (xMgr *xManager) getSrvWatchFlag() bool {
	return xMgr.needSrvWatch
}

func (xMgr *xManager) setCfgWatchFlag(flag bool) {
	xMgr.needCfgWatch = flag
}

func (xMgr *xManager) getCfgWatchFlag() bool {
	return xMgr.needCfgWatch
}

func (xMgr *xManager) setCacheAddress(addrs []string) {
	xMgr.addrCache = addrs
}

// get srv address cache
// if tSrvAddr is not equal to srv addr cache,
// need to notified ,otherwise do not need to notified.
func (xMgr *xManager) getCacheAddress(tAddr []string) ([]string, bool) {
	var needNotified bool

	if reflect.DeepEqual(tAddr, xMgr.addrCache) {
		needNotified = false
		return nil, needNotified
	} else {
		needNotified = true
		xMgr.addrCache = tAddr
		return tAddr, needNotified
	}
}

func (xMgr *xManager) setCacheDynamicCfg(cacheCfg map[string][]byte) {
	xMgr.dynamicCfgCache = cacheCfg
}

// get dynamic cfg cache
// if tDynamicCfg is not equal to dynamic config cache,
// need to notified ,otherwise do not need to notified.
func (xMgr *xManager) getCacheDynamicCfg(tDynamicCfgCache map[string][]byte) (map[string][]byte, bool) {
	var needNotified bool
	if reflect.DeepEqual(tDynamicCfgCache, xMgr.dynamicCfgCache) {
		needNotified = false
		return nil, needNotified
	} else {
		needNotified = true
		xMgr.dynamicCfgCache = tDynamicCfgCache
		return xMgr.dynamicCfgCache, needNotified
	}
}

func (xMgr *xManager) addStaticCfg(arrays []configFile) {
	for _, v := range arrays {
		xMgr.staticCfgs = append(xMgr.staticCfgs, v)
	}
}

func (xMgr *xManager) getStaticCfg() []configFile {
	return xMgr.staticCfgs
}
