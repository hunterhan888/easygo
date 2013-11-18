package main

import (
	"./apps/controller"
	web "github.com/matyhtf/easygo"
	"runtime"
	"path"
)

func main() {
	server := web.NewServer()
	
	_, __gofile__, _, _ := runtime.Caller(0)
	server.SrcRoot = path.Dir(__gofile__)
	
	server.Static("/static/", "./static/")
	server.Static("/favicon.ico", "./static/")

	server.Controller(&controller.Base{})
	
	server.Start()
}