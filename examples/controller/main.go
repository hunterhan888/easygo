package controller

import (
	web "github.com/matyhtf/easygo"
)

type MainController struct {
	web.Controller
}

func (c MainController) Index() {
	c.Assign("title", "hello easygo\n")
    c.Assign("body", "It works!\n")
    c.Render("index.php")
}

