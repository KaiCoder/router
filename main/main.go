package main

import (
	"runtime"

	"github.com/cihub/seelog"
	"../server"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	logger, err := seelog.LoggerFromConfigAsFile("seelog.xml")
	if err != nil {
		seelog.Critical("err parsing config log file", err)
		return
	}
	seelog.ReplaceLogger(logger)
	seelog.Info("start log...")

	//startCpuProfile
	/*
		f, err := os.OpenFile("./cpu.prof", os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			seelog.Error(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
		go func() {
			c := make(chan os.Signal)
			signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGKILL)
			s := <-c
			switch s {
			case syscall.SIGINT, syscall.SIGKILL:
				{
					pprof.StopCPUProfile()
					f.Close()
					seelog.Error("the program had received %v signal, will exit immediately -_-|||", s.String())
					os.Exit(1)
				}
			}
		}()

		seelog.Info("load signal detect...")
	*/
	//topic := strings.Split(cfg.MQSERVER["topics"], ",")
	//server.StartServer(cfg.MQSERVER["addr"], topic, cfg.MQSERVER["group"])
	server.StartServer()
}
