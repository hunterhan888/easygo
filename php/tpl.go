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
const SRC_PATH = "src/github.com/matyhtf/easygo"

const PHP_Engine_Script = `<?php
class GoPHP 
{
	//使用\r\n表示结束
	const DATA_EOF = "\r\n";
	const TASK_EOF = "\r\n\r\n";
	
	const MSG_OK = "OK";
	const MSG_ERR = "ERR";
	
	protected $stdin;
	protected $request_count = 1000;
	protected $root;
	protected $buffer = array();
	
	public $errCode = 0;
	public $errMsg = '';
	
	function __construct()
	{
		global $argv;
		if(empty($argv[1]))
		{
			throw new Exception("Require template path.");
		}
		$this->root = trim($argv[1]).'/';
		$this->stdin = fopen("php://stdin", "r");
	}
	
	function log($var)
	{
		if(is_string($var))
		{
			$log = $var."\n";
		}
		else 
		{
			$log = var_export($var, true)."\n";
		}
		file_put_contents('/tmp/gophp.log', $log, FILE_APPEND);
	}
	
	function onTask($taskId, $tpl)
	{
		$file = $this->root.$tpl;
		if(!empty($this->buffer[$taskId]))
		{
			if(is_array($this->buffer[$taskId]))
			{
				if(isset($this->buffer[$taskId]['.']))
				{
					extract($this->buffer[$taskId]['.']);
					unset($this->buffer[$taskId]['.']);
				}
				extract($this->buffer[$taskId]);
			}
			$this->log($this->buffer[$taskId]);
		}
		
		if(!is_file($file))
		{
			$this->errCode = 4;
			$this->errMsg = "$file not found.";
			return false;
		}
		else
		{
			include $file;
		}
		return true;
	}
	
	function fetchTask() 
	{
		$data = '';
		while(1)
		{
			$read = fread($this->stdin, 8192);
			//进程结束了
			if($read == "")
			{
				exit("worker is stop\n");
			}
			$data .= $read;
			//数据结束
			if(substr($read, -2, 2)==self::DATA_EOF)
			{
				break;
			}
		}
		return $data;
	}
	
	function mainLoop()
	{
		$data = '';
		while(1)
		{
			$this->errCode = 0;
			$task = explode("|", $this->fetchTask(), 4);
// 			$this->log($task);
			//格式解析失败
			if(count($task) < 2)
			{
				$this->errCode = 1;
				goto fail;
			}
			$taskId = $task[0];
			switch($task[1])
			{
				case 'assign':
					$json_data = json_decode(trim($task[3]), true);
					//json解析失败
					if(empty($json_data))
					{
						$this->errCode = 2;
						goto fail;
					}
					if(!isset($this->buffer[$taskId][$task[2]]))
					{
						$this->buffer[$taskId][$task[2]] = $json_data;
					}
					else 
					{
						if(is_array($json_data))
						{
							$this->buffer[$taskId][$task[2]] += $json_data;
						}
						else 
						{
							$this->buffer[$taskId][$task[2]] = $json_data;
						}
					}
// 					$this->log($this->buffer);
					$this->success();
					break;
				case 'render':
					$ret = $this->onTask($taskId, trim($task[2]));
					//业务逻辑失败
					if($ret === false)
					{
						goto fail;
					}
					//清理掉
					unset($this->buffer[$taskId]);
					echo self::TASK_EOF;
					break;
				default:
					$this->errCode = 3;
					goto fail;
			}
			
			continue;
			//$request_count--;
			fail:
			$this->fail();
			$data = '';
		}
	}
	
	function success()
	{
		echo self::MSG_OK;
	}
	
	function fail()
	{
		echo self::MSG_ERR."|CODE={$this->errCode}|MSG={$this->errMsg}".self::TASK_EOF;
	}
}

$engine = new GoPHP();
$engine->mainLoop();
exit(0);
`

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
		log.Fatal("StdinPipe Error:", err)
	}
	w.Stdout, err = w.Cmd.StdoutPipe()
	if err != nil {
		log.Fatal("StdoutPipe Error:", err)
	}
	err = w.Cmd.Start()
	if err != nil {
		log.Fatal("Start", err)
	}
	go w.Wait()
}

func (w *Worker) Wait() {
	err := w.Cmd.Wait()
	w.Engine.C <- w.Id
	if err != nil {
		log.Println("Wait Error:", err)
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
		log.Fatalln("create PHP_Engine_Script fail. error=", err)
		return
	}
	log.Println("PHP Engine Start.File=" + e.PhpFile)
	//创建worker进程
	for i := 0; i < e.WorkerNum; i++ {
		w := NewWorker(e, i)
		e.Workers[i] = w
		w.Run()
	}
}
