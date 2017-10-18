package xconf

import (
	"fmt"
	log "github.com/cihub/seelog"
	nativeLog "log"
	"os"
	"strings"
	"time"
)

// xConf log level control and log redirection
func newxConfLogger(srv string) (err error) {
	params := os.Args
	var logConfStr string
	if index := len(params); index >= 1 {
		if strings.Contains(params[index-1], "xConfDebug") {
			logConfStr = fmt.Sprintf(`<seelog type="sync" minlevel="trace">
    		<outputs formatid="main" >
    			<filter levels="info,error,critical,trace,warn,debug">
        			<rollingfile type="size" filename="xConf/log/%s/xConf.log" maxsize="10475520" maxrolls="20"/>
        		</filter>
    		</outputs>
    		`, srv) + `<formats>
        		<format id="main" format="[%Date(2006-01-02 15:04:05.000)][%LEV][%Func %File:%Line] %Msg%n"/>
        		<format id="colored"  format="%EscM(46)%Level%EscM(49) %Msg%n%EscM(0)"/>
    		</formats>
		</seelog>`
		} else {
			logConfStr = fmt.Sprintf(`<seelog type="sync" minlevel="debug">
    		<outputs formatid="main" >
    			<filter levels="info,error,warn">
        			<rollingfile type="size" filename="xConf/log/%s/xConf.log" maxsize="10475520" maxrolls="20"/>
        		</filter>
    		</outputs>
    		`, srv) + `<formats>
        		<format id="main" format="[%Date(2006-01-02 15:04:05.000)][%LEV][%Func %File:%Line] %Msg%n"/>
        		<format id="colored"  format="%EscM(46)%Level%EscM(49) %Msg%n%EscM(0)"/>
    		</formats>
		</seelog>`
		}
	}

	logger, err = log.LoggerFromConfigAsString(logConfStr)
	if err != nil {
		logger.Info("xConf log load error:", err)
	}

	logger.Infof(`=============================================================
	iFlyTEK xConf SDK file
	Subject :    iFlytek xConf SDK
	Created-Time :    %s
	PID: %d
	Version: %s
=============================================================`, time.Now().String(), os.Getpid(),version)
	defer logger.Flush()

	return err
}

func newZooLogger(srv string) (zooLogger *nativeLog.Logger) {
	zooLogFile := fmt.Sprintf("xConf/log/%s/zoo.log", srv)
	os.MkdirAll("xConf/log/"+srv, 0777)
	logFile, err := os.OpenFile(zooLogFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		logger.Warn("newZooLogger | create os error:", err)
	}

	logBeginStr := fmt.Sprintf(`=============================================================
	iFlyTEK xConf SDK zookeeper file
	Subject :    iFlytek zookeeper log
	Created-Time :    %s
	PID: %d
=============================================================`, time.Now().String(), os.Getpid())
	logFile.WriteString(logBeginStr)
	logFile.WriteString("\n")

	zooLogger = nativeLog.New(logFile, "[zooLog]", nativeLog.LstdFlags)
	return zooLogger
}
