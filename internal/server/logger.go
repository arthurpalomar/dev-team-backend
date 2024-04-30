package server

import "github.com/sadlil/gologger"

var Logger gologger.GoLogger

func SetLogger(fileLog string) {
	//logger = gologger.GetLogger(gologger.CONSOLE, gologger.SimpleLog)
	Logger = gologger.GetLogger(gologger.FILE, fileLog)
	Logger.Info("Start program")
}
