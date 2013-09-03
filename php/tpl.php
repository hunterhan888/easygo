<?php

class GoPHP 
{
	//使用\r\n表示结束
	const DATA_EOF = "\r\n";
	const TASK_EOF = "\r\n\r\n";
	
	protected $stdin;
	protected $request_count = 1000;
	protected $root;
	protected $buffer = array();
	
	public $errCode = 0;
	public $errMsg = '';
	
	function __construct()
	{
		$this->root = realpath(__DIR__.'/../../../template').'/';
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
						$this->buffer[$taskId][$task[2]] = array();
					}
					$this->buffer[$taskId][$task[2]] += $json_data;
// 					$this->log($this->buffer);
					echo 'ok';
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
			echo "ERR={$this->errCode}|MSG={$this->errMsg}".self::DATA_EOF;
			$data = '';
		}
	}
}

$engine = new GoPHP();
$engine->mainLoop();
exit(0);
