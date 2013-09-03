package seajs


//seajs工具函数
//可能全局只需要一个

import (
    "fmt"
)

type SeaContext struct {
    ModuleDir  string
    AppDir     string
}

//入口脚本生成器 (main.js)
func (sea *SeaContext) UseMainScript(name string) string {
    _t := `seajs.use("%s/%s", function(module) { module.start() }) `
    return sea.WrapMainScript(fmt.Sprintf(_t, sea.AppDir, name))
}

//seajsnode 生成器
func (sea *SeaContext) WrapMainScript(script string) string {
    return `<script
        id="seajsnode"
        type="text/javascript"
        charset="utf-8">` + script + "</script>"
}
