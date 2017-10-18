package xconf

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

type dumpHandler struct {
	dumpParentFilePath  string
	dumpStaticFilePath  string
	dumpDynamicFilePath string
	lock                *sync.RWMutex
}

var e *dumpHandler
var dumpOnce sync.Once

// new exception handler
func newDumpHandler(srv string) *dumpHandler {
	dumpOnce.Do(func() {
		e = &dumpHandler{
			dumpParentFilePath: "xConf/config/" + srv,
			lock:               new(sync.RWMutex),
		}
		e.dumpStaticFilePath = path.Join(e.dumpParentFilePath, "static.dumpfile")
		e.dumpDynamicFilePath = path.Join(e.dumpParentFilePath, "dynamic.dumpfile")
		os.MkdirAll(e.dumpParentFilePath, 0777)

	})

	return e
}

// write static dump file
func (e *dumpHandler) writeStaticDumpFile(cfgDataArray []configFile) (err error) {

	content, err := json.Marshal(cfgDataArray)
	if err != nil {
		logger.Error("xConf writeStaticDumpFile | config data array marshal error : ", err)
		return err
	}

	e.lock.Lock()
	defer e.lock.Unlock()
	err = ioutil.WriteFile(e.dumpStaticFilePath, content, 0666)
	if err != nil {
		logger.Error("xConf writeStaticDumpFile | write dump file error:", err)
	}

	return err
}

// write dynamic dump file
func (e *dumpHandler) writeDynamicDumpFile(cfgDataArray []configFile) (err error) {

	content, err := json.Marshal(cfgDataArray)
	if err != nil {
		logger.Error("xConf writeDynamicDumpFile | config data array marshal error : ", err)
		return err
	}

	e.lock.Lock()
	defer e.lock.Unlock()
	err = ioutil.WriteFile(e.dumpDynamicFilePath, content, 0666)
	if err != nil {
		logger.Error("xConf writeDynamicDumpFile | write dump file error:", err)
	}

	return err
}

//read dump file
func (e *dumpHandler) readDumpFile() (result map[string][]byte, err error) {
	logger.Trace("xConf readDumpFile")
	cfgDataArray := make([]configFile, 0)
	result = make(map[string][]byte)
	e.lock.RLock()
	defer e.lock.RUnlock()
	fStatic, _ := os.Open(e.dumpStaticFilePath)
	fDynamic, _ := os.Open(e.dumpDynamicFilePath)
	defer fStatic.Close()
	defer fDynamic.Close()

	if fStatic != nil {
		content, err := ioutil.ReadAll(fStatic)
		if err != nil {
			logger.Error("xConf ReadDumpFile | static dump file read error : ", err)
			return nil, err
		}
		tempArray := make([]configFile, 0)
		err = json.Unmarshal(content, &tempArray)
		if err != nil {
			logger.Error("xConf ReadDumpFile | static dump file content unmarshal error : ", err)
			return nil, err
		}
		for _, v := range tempArray {
			cfgDataArray = append(cfgDataArray, v)
		}
	}

	if fDynamic != nil {
		content, err := ioutil.ReadAll(fDynamic)
		if err != nil {
			logger.Error("xConf ReadDumpFile | static dump file read error : ", err)
			return nil, err
		}
		tempArray := make([]configFile, 0)
		err = json.Unmarshal(content, &tempArray)
		if err != nil {
			logger.Error("xConf ReadDumpFile | static dump file content unmarshal error : ", err)
			return nil, err
		}
		for _, v := range tempArray {
			cfgDataArray = append(cfgDataArray, v)
		}
	}

	for _, v := range cfgDataArray {
		result[v.Name] = v.Content
	}

	return result, err
}
