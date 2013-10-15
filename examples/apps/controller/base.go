package controller

import (
	"github.com/matyhtf/easygo"
)

type Base struct {
	easygo.Controller
}

func (c Base) Act_Index() {
	c.Assign("title", "hello easygo\n")
    c.Assign("body", "It works!\n")
    c.Render("index.php")
}

