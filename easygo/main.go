package main

import (
	"flag"
	"fmt"
	"os"
    "os/exec"
	"strings"
)

const header = `--------------------------------------------------
  easygo! https://github.com/matyhtf/easygo
--------------------------------------------------
`

const usageText = `usage: easygo command [arguments]

The commands are:

    new $project_name | create new project & Initialization

`

// Cribbed from the genius organization of the "go" command.
type Command struct {
	Run                    func(args []string)
	UsageLine, Short, Long string
}

func (cmd *Command) Name() string {
	name := cmd.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func cmd_NewProject(name string) {
    from_dir := os.Getenv("GOPATH") + "/src/github.com/matyhtf/easygo/examples"
    to_dir := "./" + name
	cmd := exec.Command("cp", "-r", from_dir, to_dir)
    cmd.Run()
}

func main() {
	fmt.Fprintf(os.Stdout, header)
	flag.Usage = func() { usage() }
	flag.Parse()
	args := flag.Args()

	if len(args) > 0 {
		switch(args[0]) {
			case "create":
			case "new":
                if len(args) >= 2 {
				    cmd_NewProject(args[1])
                } else {
                    fmt.Printf("command %q require project_name\n", args[0])
                }
				return
			case "help":
				usage()
				break;
			default:
				break;
		}
		fmt.Printf("unknown command %q\nRun 'easygo help' for usage.\n", args[0])
	} else {
		usage()
	}
}

func usage() {
	fmt.Println(usageText)
	os.Exit(0)
}
