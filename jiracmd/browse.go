package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	jira "github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/pkg/browser"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type BrowseOptions struct {
	Project string `yaml:"project,omitempty" json:"project,omitempty"`
	Issue   string `yaml:"issue,omitempty" json:"issue,omitempty"`
}

func CmdBrowseRegistry() *jiracli.CommandRegistryEntry {
	opts := BrowseOptions{}

	return &jiracli.CommandRegistryEntry{
		"Open issue in browser",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			cmd.Arg("ISSUE", "Issue to browse to").Required().StringVar(&opts.Issue)
			jiracli.LoadConfigs(cmd, fig, &opts)
			return nil
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			opts.Issue = jiracli.FormatIssue(opts.Issue, opts.Project)
			return CmdBrowse(globals, opts.Issue)
		},
	}
}

// CmdBrowse open the default system browser to the provided issue
func CmdBrowse(globals *jiracli.GlobalOptions, issue string) error {
	return browser.OpenURL(jira.URLJoin(globals.Endpoint.Value, "browse", issue))
}
