package jiracmd

import (
	"fmt"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type VersionsOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	Project               string `yaml:"project,omitempty" json:"project,omitempty"`
}

func CmdVersionsRegistry() *jiracli.CommandRegistryEntry {
	opts := VersionsOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("versions"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		"Show versions for a project",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdVersionsUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdVersions(o, globals, &opts)
		},
	}
}

func CmdVersionsUsage(cmd *kingpin.CmdClause, opts *VersionsOptions) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	cmd.Flag("project", "project to list versions").Short('p').StringVar(&opts.Project)

	return nil
}

// CmdVersions will get available versions for project and send to the  "versions" template
func CmdVersions(o *oreo.Client, globals *jiracli.GlobalOptions, opts *VersionsOptions) error {
	if opts.Project == "" {
		return fmt.Errorf("Project Required.")
	}
	data, err := jira.GetProjectVersions(o, globals.Endpoint.Value, opts.Project)
	if err != nil {
		return err
	}
	return opts.PrintTemplate(data)
}
