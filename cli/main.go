package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/GeertJohan/go.linenoise"
	"github.com/codegangsta/cli"

	c "github.com/jxwr/cc/cli/command"
	"github.com/jxwr/cc/cli/command/initialize"
	"github.com/jxwr/cc/cli/context"
	"gopkg.in/yaml.v1"
)

var cmds = []cli.Command{
	c.NodesCommand,
	c.ChmodCommand,
	c.FailoverCommand,
	c.TakeoverCommand,
	c.MigrateCommand,
	c.ReplicateCommand,
	c.RebalanceCommand,
	c.MeetCommand,
	c.ForgetAndResetCommand,
	c.AppInfoCommand,
}

const (
	DEFAULT_HISTORY_FILE = "/.cli_history"
	DEFAULT_CONFIG_FILE  = "/.cli_config"
)

var cmdmap = map[string]cli.Command{}

func init() {
	for _, cmd := range cmds {
		cmdmap[cmd.Name] = cmd
	}
}

func showHelp() {
	fmt.Println("List of commands:")
	for _, cmd := range cmds {
		fmt.Println("  ", cmd.Name, "-", cmd.Usage)
	}
}

type CliConf struct {
	Zkhosts     string `yaml:"zkhosts,omitempty"`
	HistoryFile string `yaml:"historyfile,omitempty"`
}

func loadConfig(filename string) (*CliConf, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	conf := &CliConf{}
	err = yaml.Unmarshal(content, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		app := cli.NewApp()
		app.Name = "cli"
		app.Usage = "init a cluster"
		app.Commands = []cli.Command{initialize.Command}
		arg := append(os.Args)
		app.Run(arg)
		os.Exit(0)
	}
	if len(os.Args) == 1 {
		fmt.Println("Usage: cli <AppName> [<Command>] or cli init")
		os.Exit(1)
	}
	//load config
	user, err := user.Current()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	conf, err := loadConfig(user.HomeDir + DEFAULT_CONFIG_FILE)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Set context
	appName := os.Args[1]
	err = context.SetApp(appName, conf.Zkhosts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if conf.HistoryFile == "" {
		conf.HistoryFile = user.HomeDir + DEFAULT_HISTORY_FILE
	}
	_, err = os.Stat(conf.HistoryFile)
	if err != nil && os.IsNotExist(err) {
		_, err = os.Create(conf.HistoryFile)
		if err != nil {
			fmt.Println(conf.HistoryFile + "create failed")
		}
	}

	// REPL
	if len(os.Args) == 2 {
		err = linenoise.LoadHistory(conf.HistoryFile)
		if err != nil {
			fmt.Println(err)
		}
		for {
			str, err := linenoise.Line(appName + "> ")
			if err != nil {
				if err == linenoise.KillSignalError {
					os.Exit(1)
				}
				fmt.Printf("Unexpected error: %s\n", err)
				os.Exit(1)
			}
			fields := strings.Fields(str)

			linenoise.AddHistory(str)
			err = linenoise.SaveHistory(conf.HistoryFile)
			if err != nil {
				fmt.Println(err)
			}

			if len(fields) == 0 {
				continue
			}

			switch fields[0] {
			case "help":
				showHelp()
				continue
			case "quit":
				os.Exit(0)
			}

			cmd, ok := cmdmap[fields[0]]
			if !ok {
				fmt.Println("Error: unknown command.")
			}
			app := cli.NewApp()
			app.Name = cmd.Name
			app.Commands = []cli.Command{cmd}
			app.Run(append(os.Args[:1], fields...))
		}
	}

	// Command line
	if len(os.Args) > 2 {
		app := cli.NewApp()
		app.Name = "cli"
		app.Usage = "redis cluster cli"
		app.Commands = cmds
		app.Run(append(os.Args[:1], os.Args[2:]...))
	}
}
