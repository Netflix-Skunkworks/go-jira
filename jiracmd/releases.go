package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type ReleasesOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	Project               string `yaml:"project,omitempty" json:"project,omitempty"`
	Status                []string
	Query                 string
	OrderBy               string
}

func CmdReleasesRegistry() *jiracli.CommandRegistryEntry {
	opts := ReleasesOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("releases"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		Help: "List project releases",
		UsageFunc: func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdReleasesUsage(cmd, &opts)
		},
		ExecuteFunc: func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdReleases(o, globals, &opts)
		},
	}
}

func CmdReleasesUsage(cmd *kingpin.CmdClause, opts *ReleasesOptions) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	cmd.Flag("query", "filter the results using a literal string").Short('q').StringVar(&opts.Query)
	cmd.Flag("status", "list of status values used to filter the results by version status").Short('s').StringsVar(&opts.Status)
	cmd.Flag("order", "order the results by a field: description, name, releaseDate, sequence, startDate").StringVar(&opts.OrderBy)
	cmd.Arg("PROJECT", "project id or key").Required().StringVar(&opts.Project)
	return nil
}

func CmdReleases(o *oreo.Client, globals *jiracli.GlobalOptions, opts *ReleasesOptions) error {
	data, err := jira.GetProjectVersionsPaginated(o, globals.Endpoint.Value, opts.Project, opts.Status, opts.Query, opts.OrderBy)
	if err != nil {
		return err
	}
	if err := opts.PrintTemplate(data); err != nil {
		return err
	}
	return nil
}
