easygo
======

easygo是一个goweb框架，它的特点是使用php来渲染模板，go负责数据逻辑。

安装EasyGo
-----
```
go get github.com/go-sql-driver/mysql 
go get github.com/lunny/xorm
go get github.com/matyhtf/easygo/easygo
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
cd TestGo
```

