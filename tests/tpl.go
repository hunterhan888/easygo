package main

import (
	"../php"
	"time"
)

func main() {
	PHP := php.NewEngine(4, "/usr/bin/php", "./template")
	PHP.Init()
	go PHP.EngineLoop()
	time.Sleep(1e12)
}
