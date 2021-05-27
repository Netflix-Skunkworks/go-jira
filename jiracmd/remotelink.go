package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type RemoteLinkOptions struct {
	jiracli.CommonOptions      `yaml:",inline" json:",inline" figtree:",inline"`
	Issue                      string `yaml:"issue,omitempty" json:"issue,omitempty"`
	Project                    string `yaml:"project,omitempty" json:"project,omitempty"`
        // There is no existing jiradata definition for RemoteObject
        URL                        string
        Title                      string
}

func CmdRemoteLinkRegistry() *jiracli.CommandRegistryEntry {
	opts := RemoteLinkOptions {}
	return &jiracli.CommandRegistryEntry{
		"Link an issue to a remote URI",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdRemoteLinkUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			opts.Issue = jiracli.FormatIssue(opts.Issue, opts.Project)
                        return CmdRemoteLink(o, globals, &opts)
		},
	}
}

func CmdRemoteLinkUsage(cmd *kingpin.CmdClause, opts *RemoteLinkOptions) error {
	jiracli.BrowseUsage(cmd, &opts.CommonOptions)
	jiracli.EditorUsage(cmd, &opts.CommonOptions)
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)

	cmd.Arg("ISSUE", "issue").Required().StringVar(&opts.Issue)
	cmd.Arg("TITLE", "Link title").Required().StringVar(&opts.Title)
	cmd.Arg("URL", "Link URL").Required().StringVar(&opts.URL)

	return nil
}

func CmdRemoteLink(o *oreo.Client, globals *jiracli.GlobalOptions, opts *RemoteLinkOptions) error {
	if err := jira.LinkRemoteIssue(o, globals.Endpoint.Value, opts.Issue, opts.URL, opts.Title); err != nil {
		return err
	}
	// unhandled if !globals.Quiet.Value {
        // unhandled if opts.Browse.Value {

	return nil
}
