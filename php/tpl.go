package php

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"os/exec"
	//	"runtime"
	"strconv"
	"strings"
	"sync"
)

const DATA_EOF = "\r\n"
const TASK_EOF = "\r\n\r\n"

type Worker struct {
	Id, TaskN int
	Cmd       *exec.Cmd
	Stdout    io.ReadCloser
	Stdin     io.WriteCloser
	Engine    *Engine
	sync.Mutex
}

type Engine struct {
	WorkerNum       int
	PhpCli, TplPath string
	RunDir, PhpFile string //PHP会写入到一个临时文件中
	Workers         []*Worker
	C               chan int
}

type Task struct {
	Worker *Worker
	Id     int
}

type TaskError struct {
	Msg  string
	Code int
}

func NewTask(e *Engine) *Task {
	t := new(Task)
	//随即选取一个引擎
	EngineId := rand.Intn(e.WorkerNum)
	w := e.Workers[EngineId]
	t.Worker = w

	w.Lock()
	//分配一个taskId
	if w.TaskN > 210000000 {
		w.TaskN = 0
	} else {
		w.TaskN++
	}
	t.Id = w.TaskN
	w.Unlock()
	return t
}

func (e TaskError) Error() string {
	return e.Msg
}

func NewWorker(e *Engine, id int) *Worker {
	w := new(Worker)
	w.Id = id
	w.Engine = e
	return w
}

func (t *Task) Assign(name string, data interface{}) error {
	var (
		err     error
		n       int
		jsonStr []byte
		sendStr string
	)
	jsonStr, err = json.Marshal(data)
	if err != nil {
		return err
	}
	sendStr = strconv.Itoa(t.Id) + "|assign|" + name + "|" + string(jsonStr) + DATA_EOF
	t.Worker.Lock()
	defer t.Worker.Unlock()
	_, err = t.Worker.Stdin.Write([]byte(sendStr))
	if err != nil {
		return err
	}
	ret := make([]byte, 8192)
	n, err = t.Worker.Stdout.Read(ret)

	if err != nil || string(ret[0:n]) != "OK" {
		return TaskError{
			Msg: string(ret[0:n]),
		}
	}
	return nil
}

func (t *Task) Render(tpl string) (string, error) {
	t.Worker.Lock()
	defer t.Worker.Unlock()
	var retString string = ""
	var err error
	_, err = t.Worker.Stdin.Write([]byte(strconv.Itoa(t.Id) + "|render|" + tpl + DATA_EOF))
	if err != nil {
		return "", err
	}

	var ret []byte = make([]byte, 8192)
	n, l := 0, 0
	for {
		n, err = t.Worker.Stdout.Read(ret)
		if err != nil {
			if len(retString) == 0 {
				return "", err
			}
		}
		retString += string(ret[:n])
		l = len(retString)
		if l > 4 && (retString[l-4:l] == TASK_EOF) {
			break
		}
	}
	return strings.TrimSpace(retString), nil
}

func NewEngine(worker_num int, php_cli, tpl_path string) *Engine {
	tpl := new(Engine)
	tpl.WorkerNum = worker_num
	tpl.PhpCli = php_cli
	tpl.C = make(chan int, 100)
	tpl.TplPath = tpl_path
	return tpl
}

/**
 * worker进程管理，挂掉的worker重新拉起
 */
func (t *Engine) EngineLoop() {
	for {
		//等待worker结束事件
		workerId := <-t.C
		//重新拉起新的Worker
		t.Workers[workerId].Run()
	}
}

func (w *Worker) Run() {
	var err error
	w.Cmd = exec.Command(w.Engine.PhpCli, "-f", w.Engine.PhpFile, w.Engine.TplPath)
	w.Stdin, err = w.Cmd.StdinPipe()
	if err != nil {
		log.Fatal("PHPEngine: StdinPipe Error:", err)
	}
	w.Stdout, err = w.Cmd.StdoutPipe()
	if err != nil {
		log.Fatal("PHPEngine: StdoutPipe Error:", err)
	}
	err = w.Cmd.Start()
	if err != nil {
		log.Fatal("PHPEngine: Start fail. Error: ", err)
	}
	go w.Wait()
}

func (w *Worker) Wait() {
	err := w.Cmd.Wait()
	w.Engine.C <- w.Id
	if err != nil {
		log.Println("PHPEngine: Wait Error:", err)
	}
}

func (e *Engine) Init() {
	e.Workers = make([]*Worker, e.WorkerNum)
	//将php文件写入到临时目录中
	if e.RunDir == "" {
		e.RunDir = "/tmp"
	}
	e.PhpFile = e.RunDir + "/GoPHP.php"
	
	err := File_put_contents(e.PhpFile, PHP_Engine_Script)
	if err!= nil {
		log.Fatalln("PHPEngine: Create PHP_Engine_Script fail. error=", err)
		return
	}
	log.Println("PHPEngine: Start.File=" + e.PhpFile)
	//创建worker进程
	for i := 0; i < e.WorkerNum; i++ {
		w := NewWorker(e, i)
		e.Workers[i] = w
		w.Run()
	}
}
