package main

import (
	"./apps/controller"
	web "github.com/matyhtf/easygo"
	"log"
	"os"
	"path"
	"runtime"
)

func main() {
	_, __gofile__, _, _ := runtime.Caller(0)
	server := &web.Server
	server.Host = "0.0.0.0:8888"
	server.Root = path.Dir(__gofile__)

	//开发环境
	if len(os.Args) > 1 && os.Args[1] == "dev" {
		server.Env = "dev"
		server.Debug = true
	} else {
		logFile, err := os.OpenFile("./server.log", os.O_RDWR|os.O_CREATE, 0666)
		if err!= nil{
			panic(err)
			return
		}
		log.SetOutput(logFile)
	}

	server.MYSQL_DSN_DEV = "root:root@tcp(localhost:3306)/test?charset=utf8"
	server.MYSQL_DSN = "root:passwd@tcp(localhost:3306)/test?charset=utf8"
	server.MYSQL_DEBUG = false

	server.SessionKey = "GOSESSID"
	server.SessionDir = "/tmp/gosess"
	server.SessionLifetime = 24 * 3600

	server.Static("/static/", "./static/")
	server.Static("/favicon.ico", "./static/")
	
	server.Controller(&controller.Base{})	
	server.Start()
}
