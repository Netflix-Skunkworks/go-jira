package main

import (
	"os"
	"path/filepath"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"gopkg.in/jira.v1/jiracli"
	"gopkg.in/jira.v1/jiracmd"
	"gopkg.in/op/go-logging.v1"
)

type oreoLogger struct {
	logger *logging.Logger
}

var log = logging.MustGetLogger("jira")

func (ol *oreoLogger) Printf(format string, args ...interface{}) {
	ol.logger.Debugf(format, args...)
}

func main() {
	defer jiracli.HandleExit()

	jiracli.InitLogging()

	configDir := ".jira.d"
	fig := figtree.NewFigTree(
		figtree.WithHome(jiracli.Homedir()),
		figtree.WithEnvPrefix("JIRA"),
		figtree.WithConfigDir(configDir),
	)

	if err := os.MkdirAll(filepath.Join(jiracli.Homedir(), configDir), 0755); err != nil {
		log.Errorf("%s", err)
		panic(jiracli.Exit{Code: 1})
	}

	o := oreo.New().WithCookieFile(filepath.Join(jiracli.Homedir(), configDir, "cookies.js")).WithLogger(&oreoLogger{log})

	jiracmd.RegisterAllCommands()

	app := jiracli.CommandLine(fig, o)
	jiracli.ParseCommandLine(app, os.Args[1:])
}
