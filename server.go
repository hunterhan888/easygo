package easygo

import (
	"fmt"
	"github.com/Unknwon/goconfig"
	_ "github.com/go-sql-driver/mysql"
	"github.com/lunny/xorm"
	"github.com/matyhtf/easygo/php"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
)

const DEFAULT_PHP_CLI = "/usr/bin/php"
const DEFAULT_PHP_WORKER_NUM = 4

var Server ServerType
var err error
var once sync.Once

type ControllerType struct {
	Type    reflect.Type
	Value   reflect.Value
	Methods map[string]int
}

type ServerType struct {
	//MVC
	Controllers       map[string]*ControllerType
	DefaultController string
	DefaultAction     string
	//server
	Host               string
	WebRoot            string
	MYSQL_DEBUG, Debug bool
	Root, Env          string
	LogFile            string

	Charset string
	//session
	SessionKey      string
	SessionDir      string
	SessionLifetime int
	//database
	MYSQL_DSN string
	DB        *xorm.Engine
	//php engine
	PHP            *php.Engine
	PHP_WORKER_NUM int
	PHP_CLI        string
	PHP_TPL_DIR    string
}

func (s *ServerType) init() {
	s.DB, err = xorm.NewEngine("mysql", s.MYSQL_DSN)
	if err != nil {
		panic(err)
	}
	s.DB.ShowSQL = s.MYSQL_DEBUG
	
	if s.PHP_CLI == "" {
		s.PHP_CLI = DEFAULT_PHP_CLI
	}

	if s.LogFile != "" {
		logFile, err := os.OpenFile(s.LogFile, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}
		log.SetOutput(logFile)
	}
	
	//worker_num为0表示不启用PHP引擎
	if s.PHP_WORKER_NUM != 0 {
		if s.PHP_TPL_DIR == "" {
			s.PHP_TPL_DIR = s.Root + "/static/template/"
		}
		
		//php模板引擎
		s.PHP = php.NewEngine(s.PHP_WORKER_NUM, s.PHP_CLI, s.PHP_TPL_DIR)
		//设置PHP的运行路径
		s.PHP.RunDir = s.Root
		//初始化
		s.PHP.Init()
	
		go s.PHP.EngineLoop()
	}

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

	//开发环境需要打印堆栈信息
	if s.Env != "dev" {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Error: ", err)
				if str, ok := err.(string); ok {
					http.Error(resp, str, 404)
				} else if err, ok := err.(error); ok {
					http.Error(resp, err.Error(), 404)
				}
			}
		}()
	}

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
	c, ok := s.Controllers[strings.ToLower(controllerName)]
	if !ok {
		NotFound(resp, "<h1>Http 404</h1><hr>Not Found Controller: "+controllerName)
		return
	}
	m, ok := s.Controllers[strings.ToLower(controllerName)].Methods[actionName]
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
	log.Println("EasyGo web application start. Bind", s.Host)
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
		Server.Controllers = make(map[string]*ControllerType, 10)
	})

	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c).Elem()
	p := v.FieldByName("Name")

	typeName := strings.ToLower(strings.Replace(t.String()[strings.Index(t.String(), ".")+1:], "Controller", "", 1))
	p.SetString(typeName)
	s.Controllers[typeName] = &ControllerType{
		Type:    t,
		Value:   v,
		Methods: ScanMethod(t),
	}
}

func ScanMethod(t reflect.Type) map[string]int {
	var (
		methods map[string]int = make(map[string]int, t.NumMethod())
		name    string
	)
	for i := 0; i < t.NumMethod(); i++ {
		name = t.Method(i).Name
		if len(name) > 5 && name[0:4] == "Act_" {
			methods[strings.ToLower(name[4:])] = i
		}
	}
	return methods
}

func (s *ServerType) LoadConfig(file string) error {

	conf, err := goconfig.LoadConfigFile(file)
	if err != nil {
		return err
	}

	//server
	s.Debug = conf.MustBool("server", "debug")
	s.Host = conf.MustValue("server", "host") + ":" + conf.MustValue("server", "port")
	s.WebRoot = conf.MustValue("server", "webroot")
	s.LogFile = conf.MustValue("server", "log_file")

	//database
	s.MYSQL_DSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s",
		conf.MustValue("database", "user"),
		conf.MustValue("database", "password"),
		conf.MustValue("database", "host"),
		conf.MustValue("database", "port"),
		conf.MustValue("database", "db"),
		conf.MustValue("database", "charset"))
	//sql debug
	s.MYSQL_DEBUG = conf.MustBool("database", "debug")

	//php engine
	s.PHP_WORKER_NUM = conf.MustInt("php", "worker_num")
	//或者填写绝对路径
	s.PHP_CLI = conf.MustValue("php", "cli")
	//模板文件的路径
	s.PHP_TPL_DIR = conf.MustValue("php", "tpl_dir")

	//session
	s.SessionKey = conf.MustValue("session", "key")
	s.SessionDir = conf.MustValue("session", "dir")
	s.SessionLifetime = conf.MustInt("session", "lifetime")

	return nil
}

func exit() {
	os.Exit(0)
}
