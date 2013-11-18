easygo
======

easygo是一个goweb框架，它的特点是使用php来渲染模板，go负责数据逻辑。
easygo依赖php-cli，在使用前请务必确认已经安装了php-cli

文档：<http://www4swoole.sinaapp.com/wiki/index/prid-3.html>

安装EasyGo
-----
```shell
go get github.com/go-sql-driver/mysql 
go get github.com/lunny/xorm
go get github.com/matyhtf/easygo/easygo
go get github.com/Unknwon/goconfig
```

环境配置
-----
如果已设置过GOPATH，则跳过此步骤。
```
vi ~/.bashrc
export GOPATH=$HOME/your_code_path
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
```

新建项目
-----
```
cd workspace/your_project_dir
easygo new TestGo
```

运行程序
-----
```
cd TestGo
go run main.go dev
```

打开浏览器 http://localhost:8888/


