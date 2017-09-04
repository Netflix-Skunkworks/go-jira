package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"

	"github.com/coryb/figtree"
	"github.com/coryb/kingpeon"
	"github.com/coryb/oreo"

	jira "gopkg.in/Netflix-Skunkworks/go-jira.v1"
	"gopkg.in/Netflix-Skunkworks/go-jira.v1/jiracli"
	"gopkg.in/Netflix-Skunkworks/go-jira.v1/jiracmd"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/op/go-logging.v1"
)

var (
	log           = logging.MustGetLogger("jira")
	defaultFormat = "%{color}%{time:2006-01-02T15:04:05.000Z07:00} %{level:-5s} [%{shortfile}]%{color:reset} %{message}"
)

func handleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(jiracli.Exit); ok {
			os.Exit(exit.Code)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n%s", e, debug.Stack())
			os.Exit(1)
		}
	}
}

func increaseLogLevel(verbosity int) {
	logging.SetLevel(logging.GetLevel("")+logging.Level(verbosity), "")
	if logging.GetLevel("") > logging.DEBUG {
		oreo.TraceRequestBody = true
		oreo.TraceResponseBody = true
	}
}

func main() {
	defer handleExit()
	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	format := os.Getenv("JIRA_LOG_FORMAT")
	if format == "" {
		format = defaultFormat
	}
	logging.SetBackend(
		logging.NewBackendFormatter(
			logBackend,
			logging.MustStringFormatter(format),
		),
	)
	if os.Getenv("JIRA_DEBUG") == "" {
		logging.SetLevel(logging.NOTICE, "")
	} else {
		logging.SetLevel(logging.DEBUG, "")
	}

	app := kingpin.New("jira", "Jira Command Line Interface")
	app.Command("version", "Prints version").PreAction(func(*kingpin.ParseContext) error {
		fmt.Println(jira.VERSION)
		panic(jiracli.Exit{Code: 0})
	})

	var verbosity int
	app.Flag("verbose", "Increase verbosity for debugging").Short('v').PreAction(func(_ *kingpin.ParseContext) error {
		os.Setenv("JIRA_DEBUG", fmt.Sprintf("%d", verbosity))
		increaseLogLevel(1)
		return nil
	}).CounterVar(&verbosity)

	if os.Getenv("JIRA_DEBUG") != "" {
		if verbosity, err := strconv.Atoi(os.Getenv("JIRA_DEBUG")); err == nil {
			increaseLogLevel(verbosity)
		}
	}

	fig := figtree.NewFigTree()
	fig.EnvPrefix = "JIRA"
	fig.ConfigDir = ".jira.d"

	if err := os.MkdirAll(filepath.Join(jiracli.Homedir(), fig.ConfigDir), 0755); err != nil {
		log.Errorf("%s", err)
		panic(jiracli.Exit{Code: 1})
	}

	o := oreo.New().WithCookieFile(filepath.Join(jiracli.Homedir(), fig.ConfigDir, "cookies.js"))
	o = o.WithPostCallback(
		func(req *http.Request, resp *http.Response) (*http.Response, error) {
			authUser := resp.Header.Get("X-Ausername")
			if authUser == "" || authUser == "anonymous" {
				// we are not logged in, so force login now by running the "login" command
				app.Parse([]string{"login"})
				return o.Do(req)
			}
			return resp, nil
		},
	)

	registry := []jiracli.CommandRegistry{
		jiracli.CommandRegistry{
			Command: "login",
			Entry:   jiracmd.CmdLoginRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "logout",
			Entry:   jiracmd.CmdLogoutRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "list",
			Aliases: []string{"ls"},
			Entry:   jiracmd.CmdListRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "view",
			Entry:   jiracmd.CmdViewRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "create",
			Entry:   jiracmd.CmdCreateRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "edit",
			Entry:   jiracmd.CmdEditRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "comment",
			Entry:   jiracmd.CmdCommentRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "worklog list",
			Entry:   jiracmd.CmdWorklogListRegistry(o),
			Default: true,
		},
		jiracli.CommandRegistry{
			Command: "worklog add",
			Entry:   jiracmd.CmdWorklogAddRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "fields",
			Entry:   jiracmd.CmdFieldsRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "createmeta",
			Entry:   jiracmd.CmdCreateMetaRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "editmeta",
			Entry:   jiracmd.CmdEditMetaRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "subtask",
			Entry:   jiracmd.CmdSubtaskRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "dup",
			Entry:   jiracmd.CmdDupRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "block",
			Entry:   jiracmd.CmdBlockRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "issuelink",
			Entry:   jiracmd.CmdIssueLinkRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "issuelinktypes",
			Entry:   jiracmd.CmdIssueLinkTypesRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "transition",
			Aliases: []string{"trans"},
			Entry:   jiracmd.CmdTransitionRegistry(o, ""),
		},
		jiracli.CommandRegistry{
			Command: "transitions",
			Entry:   jiracmd.CmdTransitionsRegistry(o, "transitions"),
		},
		jiracli.CommandRegistry{
			Command: "transmeta",
			Entry:   jiracmd.CmdTransitionsRegistry(o, "debug"),
		},
		jiracli.CommandRegistry{
			Command: "close",
			Entry:   jiracmd.CmdTransitionRegistry(o, "close"),
		},
		jiracli.CommandRegistry{
			Command: "acknowledge",
			Aliases: []string{"ack"},
			Entry:   jiracmd.CmdTransitionRegistry(o, "acknowledge"),
		},
		jiracli.CommandRegistry{
			Command: "reopen",
			Entry:   jiracmd.CmdTransitionRegistry(o, "reopen"),
		},
		jiracli.CommandRegistry{
			Command: "resolve",
			Entry:   jiracmd.CmdTransitionRegistry(o, "resolve"),
		},
		jiracli.CommandRegistry{
			Command: "start",
			Entry:   jiracmd.CmdTransitionRegistry(o, "start"),
		},
		jiracli.CommandRegistry{
			Command: "stop",
			Entry:   jiracmd.CmdTransitionRegistry(o, "stop"),
		},
		jiracli.CommandRegistry{
			Command: "todo",
			Entry:   jiracmd.CmdTransitionRegistry(o, "To Do"),
		},
		jiracli.CommandRegistry{
			Command: "backlog",
			Entry:   jiracmd.CmdTransitionRegistry(o, "Backlog"),
		},
		jiracli.CommandRegistry{
			Command: "done",
			Entry:   jiracmd.CmdTransitionRegistry(o, "Done"),
		},
		jiracli.CommandRegistry{
			Command: "in-progress",
			Aliases: []string{"prog", "progress"},
			Entry:   jiracmd.CmdTransitionRegistry(o, "Progress"),
		},
		jiracli.CommandRegistry{
			Command: "vote",
			Entry:   jiracmd.CmdVoteRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "rank",
			Entry:   jiracmd.CmdRankRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "watch",
			Entry:   jiracmd.CmdWatchRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "labels add",
			Entry:   jiracmd.CmdLabelsAddRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "labels set",
			Entry:   jiracmd.CmdLabelsSetRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "labels remove",
			Entry:   jiracmd.CmdLabelsRemoveRegistry(o),
			Aliases: []string{"rm"},
		},
		jiracli.CommandRegistry{
			Command: "take",
			Entry:   jiracmd.CmdTakeRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "assign",
			Entry:   jiracmd.CmdAssignRegistry(o),
			Aliases: []string{"give"},
		},
		jiracli.CommandRegistry{
			Command: "unassign",
			Entry:   jiracmd.CmdUnassignRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "component add",
			Entry:   jiracmd.CmdComponentAddRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "components",
			Entry:   jiracmd.CmdComponentsRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "issuetypes",
			Entry:   jiracmd.CmdIssueTypesRegistry(o),
		},
		jiracli.CommandRegistry{
			Command: "export-templates",
			Entry:   jiracmd.CmdExportTemplatesRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "unexport-templates",
			Entry:   jiracmd.CmdUnexportTemplatesRegistry(),
		},
		jiracli.CommandRegistry{
			Command: "browse",
			Entry:   jiracmd.CmdBrowseRegistry(),
			Aliases: []string{"b"},
		},
		jiracli.CommandRegistry{
			Command: "request",
			Entry:   jiracmd.CmdRequestRegistry(o),
			Aliases: []string{"req"},
		},
	}

	jiracli.Register(app, fig, registry)

	// register custom commands
	data := struct {
		CustomCommands kingpeon.DynamicCommands `yaml:"custom-commands" json:"custom-commands"`
	}{}

	if err := fig.LoadAllConfigs("config.yml", &data); err != nil {
		log.Errorf("%s", err)
		panic(jiracli.Exit{Code: 1})
	}

	if len(data.CustomCommands) > 0 {
		tmp := map[string]interface{}{}
		fig.LoadAllConfigs("config.yml", &tmp)
		kingpeon.RegisterDynamicCommands(app, data.CustomCommands, jiracli.TemplateProcessor())
	}

	app.Terminate(func(status int) {
		for _, arg := range os.Args {
			if arg == "-h" || arg == "--help" || len(os.Args) == 1 {
				panic(jiracli.Exit{Code: 0})
			}
		}
		panic(jiracli.Exit{Code: 1})
	})

	// checking for default usage of `jira ISSUE-123` but need to allow
	// for global options first like: `jira --user mothra ISSUE-123`
	ctx, _ := app.ParseContext(os.Args[1:])
	if ctx != nil {
		if ctx.SelectedCommand == nil {
			next := ctx.Next()
			if next != nil {
				if ok, err := regexp.MatchString("^[A-Z]+-[0-9]+$", next.Value); err != nil {
					log.Errorf("Invalid Regex: %s", err)
				} else if ok {
					// insert "view" at i=1 (2nd position)
					os.Args = append(os.Args[:1], append([]string{"view"}, os.Args[1:]...)...)
				}
			}
		}
	}

	if _, err := app.Parse(os.Args[1:]); err != nil {
		if _, ok := err.(*jiracli.Error); ok {
			log.Errorf("%s", err)
			panic(jiracli.Exit{Code: 1})
		} else {
			ctx, _ := app.ParseContext(os.Args[1:])
			if ctx != nil {
				app.UsageForContext(ctx)
			}
			log.Errorf("Invalid Usage: %s", err)
			panic(jiracli.Exit{Code: 1})
		}
	}
}
