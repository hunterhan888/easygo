package easygo

import (
	"github.com/matyhtf/easygo/php"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lunny/xorm"
	"log"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
)

var Server ServerType
var err error
var once sync.Once

type ControllerType struct {
	Type reflect.Type
	Value reflect.Value
	Methods map[string]int
}

type ServerType struct {
	controllers        map[string] *ControllerType
	Host               string
	MYSQL_DEBUG, Debug bool
	Root, Env          string
	DB                 *xorm.Engine
	SessionKey         string
	SessionDir         string
	SessionLifetime    int
	PHP                *php.Engine
	MYSQL_DSN          string
	MYSQL_DSN_DEV      string
	
	Charset string
	
	DefaultController  string
	DefaultAction      string
}

func (s *ServerType) init() {
	//开发环境使用LOCAL数据库
	if s.Env == "dev" {
		s.DB, err = xorm.NewEngine("mysql", s.MYSQL_DSN_DEV)
	} else {
		s.DB, err = xorm.NewEngine("mysql", s.MYSQL_DSN)
	}
	if err != nil {
		panic(err)
	}
	err = s.DB.SetPool(xorm.NewSimpleConnectPool())
	if err != nil {
		panic(err)
	}
	s.DB.ShowSQL = s.MYSQL_DEBUG
	//php模板引擎
	s.PHP = php.NewEngine(1, "/usr/bin/php", s.Root + "/apps/template")
	s.PHP.Init()
	
	go s.PHP.EngineLoop()
	go Session_CheckExpire()
	
	//默认
	if s.DefaultAction == "" {
		s.DefaultAction = "index"
	}
	if s.DefaultController == "" {
		s.DefaultController = "Base"
	}
	s.Charset = "utf-8"
}

func (s *ServerType) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	//	defer func() {
	//		if err := recover(); err != nil {
	//			log.Println("Error: ", err)
	//			if str, ok := err.(string); ok {
	//				http.Error(resp, str, 404)
	//			} else if err, ok := err.(error); ok {
	//				http.Error(resp, err.Error(), 404)
	//			}
	//		}
	//	}()
	
	log.Println(req.Method, req.RemoteAddr, req.URL.Path)
	
	var controllerName, actionName string
	path := php.Explode("/", strings.Trim(strings.ToLower(req.URL.Path), " /"), 3)
	
	if len(path) == 1 {
		if path[0] == "" {
			controllerName = s.DefaultController
		} else {
			controllerName = path[0]
		}
		actionName = s.DefaultAction
	} else {
		controllerName = path[0]
		actionName = path[1]
	}
	//Invoke the request handler
	c, ok := s.controllers[strings.ToLower(controllerName)]
	if !ok {
		NotFound(resp, "<h1>Http 404</h1><hr>Not Found Controller: " + controllerName)
		return
	}
	m, ok := s.controllers[strings.ToLower(controllerName)].Methods[actionName]
	if !ok {
		NotFound(resp, fmt.Sprintf("<h1>Http 404</h1><hr>Not Found Action: %s->%s\n", controllerName, actionName))
		return
	}
	
//	log.Println(c.Methods)
	vc := reflect.New(c.Value.Type())

	in := make([]reflect.Value, 3)
	in[0] = reflect.ValueOf(resp)
	in[1] = reflect.ValueOf(req)
	in[2] = reflect.ValueOf(actionName)
	
	nothing := []reflect.Value{}
	
	//初始化请求
	vc.MethodByName("OnRequest").Call(in)
	//Controller初始化
	vc.MethodByName("Init").Call(nothing)
	//Do Action
	vc.Method(m).Call(nothing)
	//Controller释放
	vc.MethodByName("Destroy").Call(nothing)
	//释放请求
	vc.MethodByName("OnFinish").Call(nothing)
}

func (s *ServerType) Start() {
	log.Println("Go web application erver start.", s.Host)
	s.init()
	http.Handle("/", s)
	err := http.ListenAndServe(s.Host, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (s *ServerType) Static(path, dir string) {
	http.Handle(path, http.StripPrefix(path, http.FileServer(http.Dir(dir))))
}

func (s *ServerType) Controller(c interface{}) {
	once.Do(func() {
		Server.controllers = make(map[string] *ControllerType, 10)
	})
	
	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c).Elem()
	p := v.FieldByName("Name")
	
	typeName := strings.ToLower(strings.Replace(t.String()[strings.Index(t.String(), ".")+1:], "Controller", "", 1))
	p.SetString(typeName)
	s.controllers[typeName] = &ControllerType{
		Type: t,
		Value: v,
		Methods: ScanMethod(t),
	}
}

func ScanMethod(t reflect.Type) map[string]int {
	var (
		methods map[string]int = make(map[string]int , t.NumMethod())
		name string
	)
	for i:=0; i<t.NumMethod(); i++ {
		name = t.Method(i).Name
		if len(name) > 5 && name[0:4] == "Act_" {
			methods[strings.ToLower(name[4:])] = i
		}
	}
	return methods
}

func exit() {
	os.Exit(0)
}
