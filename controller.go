package easygo

import (
	"fmt"
	"html/template"
	"io/ioutil"
	//	"strconv"
	"log"
	"net/http"
	//"time"
	"github.com/matyhtf/easygo/php"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lunny/xorm"

//	"strings"
)

const ERR_PARAM = 1001

//基础controller
type Layout struct {
	Fn           string
	Title        string
	HeaderScript []string
	FooterScript []string
}

type Controller struct {
	Resp        http.ResponseWriter
	Req         *http.Request
	Name        string
	Action      string
	Uid         int
	layoutFn    string
	Layout      Layout
	jsonEncoder *json.Encoder
	Session     *SessionType
	DB          *xorm.Engine
	tpl         *php.Task
}

//启动时初始化
func NewController() *Controller {
	c := new(Controller)
	//前端
	c.Layout.Fn = "template/layout/main.html.mustache"
	c.Layout.HeaderScript = []string{}
	c.jsonEncoder = json.NewEncoder(c.Resp)
	return c
}

//request初始化
func (c *Controller) OnRequest(resp http.ResponseWriter, req *http.Request, action string) {
	c.Req = req
	c.Resp = resp
	c.DB = Server.DB
	c.Action = action
	//启动session
	c.Session = NewSession(c.Req, c.Resp)
	c.tpl = php.NewTask(Server.PHP)
	c.Header("Cache-Control", "must-revalidate, private, max-age=0")
	err := c.Req.ParseForm()
	if err != nil {
		log.Println("parse form fail")
	}
}

//request结束
func (c *Controller) OnFinish() {
	c.Session.Save()
}

func (c Controller) Init() {
}

func (c Controller) Destroy() {
}

func (c Controller) IsLogin() bool {
	var flag = true
	i := c.Session.Get("userId")
	if i == "" {
		flag = false
	}
	return flag
}

func (c Controller) Redirect(url string) {
	http.Redirect(c.Resp, c.Req, url, 302)
}

func (c Controller) Display(tpl string, data interface{}) error {
	bytes, err := ioutil.ReadFile(tpl)
	if err != nil {
		return err
	}
	t, _ := template.New("").Parse(string(bytes))
	return t.Execute(c.Resp, data)
}


func (c Controller) Render(tpl string) {
	c.Header("Content-Type", "text/html;charset="+Server.Charset)
	str, err := c.tpl.Render(tpl)
	if err != nil {
		c.Echo(err.Error())
	} else {
		c.Echo(str)
	}
}

func (c Controller) Assign(name string, data interface{}) bool {
	err := c.tpl.Assign(name, data)
	if err == nil {
		return true
	} else {
		log.Println(err.Error())
		return false
	}
}

func (c *Controller) Echo(str string) {
	fmt.Fprint(c.Resp, str)
}

func (c *Controller) Message(code int, msg string) {
	c.Echo(fmt.Sprintf("{\"code\": %d, \"msg\": \"%s\"}", code, msg))
}

func (c *Controller) Header(key string, value string) {
	c.Resp.Header().Add(key, value)
}

func (c Controller) EchoJson(data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Println(err.Error())
	} else {
		c.Echo(string(b))
	}
}

//DoAction
func (c *Controller) OnAction(action string) {

}

func (c Controller) EchoApiError(m string) {
	var data = map[string]string{
		"Status": "Error",
		"Msg":    m,
	}
	b, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		c.Echo(string(b))
	}
}

func (c Controller) EchoApiData(data interface{}) {
	var dataString string
	b, err := json.Marshal(data)
	if err != nil {
		c.Echo("panic : failed encode data")
	} else {
		dataString = string(b)
	}
	var result = map[string]string{
		"Status": "Success",
		"Msg":    "",
		"data":   dataString,
	}
	r, err := json.Marshal(result)
	if err != nil {
		c.EchoApiError("panic : " + err.Error())
	} else {
		c.Echo(string(r))
	}
}

func (c Controller) RenderJson(context ...interface{}) {
	c.jsonEncoder.Encode(context)
}

func (c Controller) Form(key string) string {
	return c.Req.FormValue(key)
}

func NotFound(resp http.ResponseWriter, msg string) {
	resp.WriteHeader(http.StatusNotFound)
	resp.Header().Add("Content-Type", "text/html;charset="+Server.Charset)
	resp.Write([]byte(msg))
}
