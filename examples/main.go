package main

import (
	"./apps/controller"
	web "github.com/matyhtf/easygo"
	"os"
	"path"
	"runtime"
)

func main() {
	_, __gofile__, _, _ := runtime.Caller(0)
	server := &web.Server
	server.Root = path.Dir(__gofile__)

	//默认为生产环境
	if len(os.Args) > 1 {
		server.Env = os.Args[1]
	} else {
		server.Env = "product"
	}
	
	server.LoadConfig("./static/config/" + server.Env + ".ini")
	
	server.Static("/static/", "./static/")
	server.Static("/favicon.ico", "./static/")

	server.Controller(&controller.Base{})
	
	server.Start()
}