package jiracmd

import (
	"fmt"
	"os"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/coryb/yaml.v2"
)

type CreateOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	jiradata.IssueUpdate  `yaml:",inline" json:",inline" figtree:",inline"`
	Project               string            `yaml:"project,omitempty" json:"project,omitempty"`
	IssueType             string            `yaml:"issuetype,omitempty" json:"issuetype,omitempty"`
	Overrides             map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	SaveFile              string            `yaml:"savefile,omitempty" json:"savefile,omitempty"`
}

func CmdCreateRegistry() *jiracli.CommandRegistryEntry {
	opts := CreateOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("create"),
		},
		Overrides: map[string]string{},
	}

	return &jiracli.CommandRegistryEntry{
		"Create issue",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdCreateUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdCreate(o, globals, &opts)
		},
	}
}

func CmdCreateUsage(cmd *kingpin.CmdClause, opts *CreateOptions) error {
	jiracli.BrowseUsage(cmd, &opts.CommonOptions)
	jiracli.EditorUsage(cmd, &opts.CommonOptions)
	jiracli.FileUsage(cmd, &opts.CommonOptions)
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	cmd.Flag("noedit", "Disable opening the editor").SetValue(&opts.SkipEditing)
	cmd.Flag("project", "project to create issue in").Short('p').StringVar(&opts.Project)
	cmd.Flag("issuetype", "issuetype in to create").Short('i').StringVar(&opts.IssueType)
	cmd.Flag("comment", "Comment message for issue").Short('m').PreAction(func(ctx *kingpin.ParseContext) error {
		opts.Overrides["comment"] = jiracli.FlagValue(ctx, "comment")
		return nil
	}).String()
	cmd.Flag("override", "Set issue property").Short('o').StringMapVar(&opts.Overrides)
	cmd.Flag("saveFile", "Write issue as yaml to file").StringVar(&opts.SaveFile)
	return nil
}

// CmdCreate sends the create-metadata to the "create" template for editing, then
// will parse the edited document as YAML and submit the document to jira.
func CmdCreate(o *oreo.Client, globals *jiracli.GlobalOptions, opts *CreateOptions) error {
	type templateInput struct {
		Meta      *jiradata.IssueType `yaml:"meta" json:"meta"`
		Overrides map[string]string   `yaml:"overrides" json:"overrides"`
	}

	if err := defaultIssueType(o, globals.Endpoint.Value, &opts.Project, &opts.IssueType); err != nil {
		return err
	}
	createMeta, err := jira.GetIssueCreateMetaIssueType(o, globals.Endpoint.Value, opts.Project, opts.IssueType)
	if err != nil {
		return err
	}

	issueUpdate := jiradata.IssueUpdate{}
	input := templateInput{
		Meta:      createMeta,
		Overrides: opts.Overrides,
	}
	input.Overrides["project"] = opts.Project
	input.Overrides["issuetype"] = opts.IssueType
	input.Overrides["user"] = globals.User.Value

	var issueResp *jiradata.IssueCreateResponse
    var fnameOptsFile string
    fnameOptsFile = opts.File.String()
    if fnameOptsFile != "" {
        err = jiracli.ReadYmlInputFile(&opts.CommonOptions, &input, &issueUpdate, func() error {
            issueResp, err = jira.CreateIssue(o, globals.Endpoint.Value, &issueUpdate)
            return err
        })
    } else {
        err = jiracli.EditLoop(&opts.CommonOptions, &input, &issueUpdate, func() error {
            issueResp, err = jira.CreateIssue(o, globals.Endpoint.Value, &issueUpdate)
            return err
        })
    }
	if err != nil {
		return err
	}

	browseLink := jira.URLJoin(globals.Endpoint.Value, "browse", issueResp.Key)
	if !globals.Quiet.Value {
		fmt.Printf("OK %s %s\n", issueResp.Key, browseLink)
	}

	if opts.SaveFile != "" {
		fh, err := os.Create(opts.SaveFile)
		if err != nil {
			return err
		}
		defer fh.Close()
		out, err := yaml.Marshal(map[string]string{
			"issue": issueResp.Key,
			"link":  browseLink,
		})
		if err != nil {
			return err
		}
		fh.Write(out)
	}

	if opts.Browse.Value {
		return CmdBrowse(globals, issueResp.Key)
	}
	return nil
}

func defaultIssueType(o *oreo.Client, endpoint string, project, issuetype *string) error {
	if project == nil || *project == "" {
		return fmt.Errorf("Project undefined, please use --project argument or set the `project` config property")
	}
	if issuetype != nil && *issuetype != "" {
		return nil
	}
	projectMeta, err := jira.GetIssueCreateMetaProject(o, endpoint, *project)
	if err != nil {
		return err
	}

	issueTypes := map[string]bool{}

	for _, issuetype := range projectMeta.IssueTypes {
		issueTypes[issuetype.Name] = true
	}

	//  prefer "Bug" type
	if _, ok := issueTypes["Bug"]; ok {
		*issuetype = "Bug"
		return nil
	}
	// next best default it "Task"
	if _, ok := issueTypes["Task"]; ok {
		*issuetype = "Task"
		return nil
	}

	return fmt.Errorf("Unable to find default issueType of Bug or Task, please set --issuetype argument or set the `issuetype` config property")
}
