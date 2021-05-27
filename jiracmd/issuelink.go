package jiracmd

import (
	"fmt"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type IssueLinkOptions struct {
	jiracli.CommonOptions     `yaml:",inline" json:",inline" figtree:",inline"`
	jiradata.LinkIssueRequest `yaml:",inline" json:",inline" figtree:",inline"`
	LinkType                  string `yaml:"linktype,omitempty" json:"linktype,omitempty"`
	Project                   string `yaml:"project,omitempty" json:"project,omitempty"`
}

func CmdIssueLinkRegistry() *jiracli.CommandRegistryEntry {
	opts := IssueLinkOptions{
		LinkIssueRequest: jiradata.LinkIssueRequest{
			Type:         &jiradata.IssueLinkType{},
			InwardIssue:  &jiradata.IssueRef{},
			OutwardIssue: &jiradata.IssueRef{},
		},
	}
	return &jiracli.CommandRegistryEntry{
		"Link two issues",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdIssueLinkUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			opts.OutwardIssue.Key = jiracli.FormatIssue(opts.OutwardIssue.Key, opts.Project)
			opts.InwardIssue.Key = jiracli.FormatIssue(opts.InwardIssue.Key, opts.Project)
			return CmdIssueLink(o, globals, &opts)
		},
	}
}

func CmdIssueLinkUsage(cmd *kingpin.CmdClause, opts *IssueLinkOptions) error {
	jiracli.BrowseUsage(cmd, &opts.CommonOptions)
	jiracli.EditorUsage(cmd, &opts.CommonOptions)
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	cmd.Flag("comment", "Comment message when linking issue").Short('m').PreAction(func(ctx *kingpin.ParseContext) error {
		opts.Comment = &jiradata.Comment{
			Body: jiracli.FlagValue(ctx, "comment"),
		}
		return nil
	}).String()
	cmd.Arg("OUTWARDISSUE", "outward issue").Required().StringVar(&opts.OutwardIssue.Key)
	cmd.Arg("ISSUELINKTYPE", "issue link type").Required().StringVar(&opts.Type.Name)
	cmd.Arg("INWARDISSUE", "inward issue").Required().StringVar(&opts.InwardIssue.Key)
	return nil
}

func CmdIssueLink(o *oreo.Client, globals *jiracli.GlobalOptions, opts *IssueLinkOptions) error {
	if err := jira.LinkIssues(o, globals.Endpoint.Value, &opts.LinkIssueRequest); err != nil {
		return err
	}

	if !globals.Quiet.Value {
		fmt.Printf("OK %s %s\n", opts.InwardIssue.Key, jira.URLJoin(globals.Endpoint.Value, "browse", opts.InwardIssue.Key))
		fmt.Printf("OK %s %s\n", opts.OutwardIssue.Key, jira.URLJoin(globals.Endpoint.Value, "browse", opts.OutwardIssue.Key))
	}

	if opts.Browse.Value {
		if err := CmdBrowse(globals, opts.OutwardIssue.Key); err != nil {
			return CmdBrowse(globals, opts.InwardIssue.Key)
		}
	}

	return nil
}
