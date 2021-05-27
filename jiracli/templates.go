package jiracli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	yaml "gopkg.in/coryb/yaml.v2"

	"github.com/Masterminds/sprig"
	"github.com/coryb/figtree"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/mgutz/ansi"
	wordwrap "github.com/mitchellh/go-wordwrap"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/crypto/ssh/terminal"
)

func findTemplate(name string) ([]byte, error) {
	if file, err := findClosestParentPath(filepath.Join(".jira.d", "templates", name)); err == nil {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	return nil, nil
}

func getTemplate(name string) (string, error) {
	if _, err := os.Stat(".jira.d/" + name); err == nil {
		b, err := ioutil.ReadFile(".jira.d/" + name)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	b, err := findTemplate(name)
	if err != nil {
		return "", err
	}
	if b != nil {
		return string(b), nil
	}
	if s, ok := AllTemplates[name]; ok {
		return s, nil
	}
	return "", fmt.Errorf("No Template found for %q", name)
}

func tmpTemplate(templateName string, data interface{}) (string, error) {
	tmpFile, err := tmpYml(templateName)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	return tmpFile.Name(), RunTemplate(templateName, data, tmpFile)
}

func TemplateProcessor() *template.Template {
	funcs := map[string]interface{}{
		"jira": func() string {
			return os.Args[0]
		},
		"env": func() map[string]string {
			out := map[string]string{}
			for _, env := range os.Environ() {
				kv := strings.SplitN(env, "=", 2)
				out[kv[0]] = kv[1]
			}
			return out
		},
		"fit": func(size int, content string) string {
			return fmt.Sprintf(fmt.Sprintf("%%-%d.%ds", size, size), content)
		},
		"shellquote": func(content string) string {
			return shellquote.Join(content)
		},
		"toMinJson": func(content interface{}) (string, error) {
			bytes, err := json.Marshal(content)
			if err != nil {
				return "", err
			}
			return string(bytes), nil
		},
		"toJson": func(content interface{}) (string, error) {
			bytes, err := json.MarshalIndent(content, "", "    ")
			if err != nil {
				return "", err
			}
			return string(bytes), nil
		},
		"termWidth": func() int {
			w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				return w
			}
			if os.Getenv("COLUMNS") != "" {
				w, err = strconv.Atoi(os.Getenv("COLUMNS"))
			}
			if err == nil {
				return w
			}
			return 120
		},
		"pctOf": func(size, percent int) int {
			return int(float32(size) * (float32(percent) / 100))
		},
		"pctVal": func(value, size int) int {
			return (size - value) * 100 / size
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"mult": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			return a / b
		},
		"append": func(more string, content interface{}) (string, error) {
			switch value := content.(type) {
			case string:
				return string(append([]byte(content.(string)), []byte(more)...)), nil
			case []byte:
				return string(append(content.([]byte), []byte(more)...)), nil
			default:
				return "", fmt.Errorf("Unknown type: %s", value)
			}
		},
		"indent": func(spaces int, content string) string {
			indent := make([]rune, spaces+1)
			indent[0] = '\n'
			for i := 1; i < spaces+1; i++ {
				indent[i] = ' '
			}

			lineSeps := []rune{'\n', '\u0085', '\u2028', '\u2029'}
			for _, sep := range lineSeps {
				indent[0] = sep
				content = strings.Replace(content, string(sep), string(indent), -1)
			}
			return content

		},
		"comment": func(content string) string {
			lineSeps := []rune{'\n', '\u0085', '\u2028', '\u2029'}
			for _, sep := range lineSeps {
				content = strings.Replace(content, string(sep), string([]rune{sep, '#', ' '}), -1)
			}
			return content
		},
		"color": func(color string) string {
			return ansi.ColorCode(color)
		},
		"remLineBreak": func(content string) string {
			return strings.Replace(strings.Replace(content, string('\r'), string(' '), -1), string('\n'), string(' '), -1)
		},
		"regReplace": func(search string, replace string, content string) string {
			re := regexp.MustCompile(search)
			return re.ReplaceAllString(content, replace)
		},
		"split": func(sep string, content string) []string {
			return strings.Split(content, sep)
		},
		"join": func(sep string, content []interface{}) string {
			vals := make([]string, len(content))
			for i, v := range content {
				vals[i] = v.(string)
			}
			return strings.Join(vals, sep)
		},
		"abbrev": func(max int, content string) string {
			if len(content) > max && max > 2 {
				var buffer bytes.Buffer
				buffer.WriteString(content[:max-3])
				buffer.WriteString("...")
				return buffer.String()
			}
			return content
		},
		"rep": func(count int, content string) string {
			var buffer bytes.Buffer
			for i := 0; i < count; i++ {
				buffer.WriteString(content)
			}
			return buffer.String()
		},
		"age": func(content string) (string, error) {
			return fuzzyAge(content)
		},
		"dateFormat": func(format string, content string) (string, error) {
			return dateFormat(format, content)
		},
		"wrap": func(width uint, content string) string {
			return wordwrap.WrapString(content, width)
		},
	}
	return template.New("gojira").Funcs(sprig.GenericFuncMap()).Funcs(funcs)
}

func ConfigTemplate(fig *figtree.FigTree, template, command string, opts interface{}) (string, error) {
	var tmp map[string]interface{}
	err := ConvertType(opts, &tmp)
	if err != nil {
		return "", err
	}
	fig.LoadAllConfigs(command+".yml", &tmp)
	fig.LoadAllConfigs("config.yml", &tmp)

	tmpl, err := TemplateProcessor().Parse(template)
	if err != nil {
		return "", err
	}
	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, &tmp); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ConvertType(input interface{}, output interface{}) error {
	// HACK HACK HACK: convert data formats to json for backwards compatibilty with templates
	jsonData, err := json.Marshal(input)
	if err != nil {
		return err
	}

	defer func(mapType, iface reflect.Type) {
		yaml.DefaultMapType = mapType
		yaml.IfaceType = iface
	}(yaml.DefaultMapType, yaml.IfaceType)

	yaml.DefaultMapType = reflect.TypeOf(map[string]interface{}{})
	yaml.IfaceType = yaml.DefaultMapType.Elem()

	if err := yaml.Unmarshal(jsonData, output); err != nil {
		return err
	}
	return nil

}

func RunTemplate(templateName string, data interface{}, out io.Writer) error {

	templateContent, err := getTemplate(templateName)
	if err != nil {
		return err
	}

	if out == nil {
		out = os.Stdout
	}

	var rawData interface{}
	err = ConvertType(data, &rawData)
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(out)
	table.SetAutoFormatHeaders(false)
	headers := []string{}
	cells := [][]string{}
	tmpl, err := TemplateProcessor().Funcs(map[string]interface{}{
		"headers": func(titles ...string) string {
			headers = append(headers, titles...)
			return ""
		},
		"row": func() string {
			cells = append(cells, []string{})
			return ""
		},
		"cell": func(value interface{}) (string, error) {
			if len(cells) == 0 {
				return "", fmt.Errorf(`"cell" template function called before "row" template function`)
			}
			cells[len(cells)-1] = append(cells[len(cells)-1], fmt.Sprintf("%v", value))
			return "", nil
		},
	}).Parse(templateContent)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(out, rawData); err != nil {
		return err
	}
	if len(headers) > 0 || len(cells) > 0 {
		table.SetHeader(headers)
		table.AppendBulk(cells)
		table.Render()
	}

	return nil
}

var AllTemplates = map[string]string{
	"attach-list":    defaultAttachListTemplate,
	"comment":        defaultCommentTemplate,
	"component-add":  defaultComponentAddTemplate,
	"components":     defaultComponentsTemplate,
	"create":         defaultCreateTemplate,
	"createmeta":     defaultDebugTemplate,
	"debug":          defaultDebugTemplate,
	"edit":           defaultEditTemplate,
	"editmeta":       defaultDebugTemplate,
	"epic-create":    defaultEpicCreateTemplate,
	"epic-list":      defaultTableTemplate,
	"fields":         defaultDebugTemplate,
	"issuelinktypes": defaultDebugTemplate,
	"issuetypes":     defaultIssuetypesTemplate,
	"json":           defaultDebugTemplate,
	"list":           defaultListTemplate,
	"request":        defaultDebugTemplate,
	"subtask":        defaultSubtaskTemplate,
	"table":          defaultTableTemplate,
	"transition":     defaultTransitionTemplate,
	"transitions":    defaultTransitionsTemplate,
	"transmeta":      defaultDebugTemplate,
	"view":           defaultViewTemplate,
	"worklog":        defaultWorklogTemplate,
	"worklogs":       defaultWorklogsTemplate,
}

const defaultDebugTemplate = "{{ . | toJson}}\n"

const defaultListTemplate = "{{ range .issues }}{{ .key | append \":\" | printf \"%-12s\"}} {{ .fields.summary }}\n{{ end }}"

const defaultTableTemplate = `{{/* table template */ -}}
{{- headers "Issue" "Summary" "Type" "Priority" "Status" "Age" "Reporter" "Assignee" -}}
{{- range .issues -}} 
  {{- row -}}
  {{- cell .key -}}
  {{- cell .fields.summary -}}
  {{- cell .fields.issuetype.name -}}
  {{- if .fields.priority -}}
    {{- cell .fields.priority.name -}}
  {{- else -}}
    {{- cell "<none>" -}}
  {{- end -}}
  {{- cell .fields.status.name -}}
  {{- cell (.fields.created | age) -}}
  {{- if .fields.reporter -}}
    {{- cell .fields.reporter.displayName -}}
  {{- else -}}
    {{- cell "<unknown>" -}}
  {{- end -}}
  {{- if .fields.assignee -}}
    {{- cell .fields.assignee.displayName -}}
  {{- else -}}
    {{- cell "<unassigned>" -}}
  {{- end -}}
{{- end -}}
`

const defaultAttachListTemplate = `{{/* attach list template */ -}}
{{- headers "id" "filename" "bytes" "user" "created" -}}
{{- range . -}}
  {{- row -}}
  {{- cell .id -}}
  {{- cell .filename -}}
  {{- cell .size -}}
  {{- cell .author.displayName -}}
  {{- cell (.created | age) -}}
{{- end -}}
`

const defaultViewTemplate = `{{/* view template */ -}}
issue: {{ .key }}
{{if .fields.created -}}
created: {{ .fields.created | age }} ago
{{end -}}
{{if .fields.status -}}
status: {{ .fields.status.name }}
{{end -}}
summary: {{ .fields.summary }}
project: {{ .fields.project.key }}
{{if .fields.components -}}
components: {{ range .fields.components }}{{ .name }} {{end}}
{{end -}}
{{if .fields.issuetype -}}
issuetype: {{ .fields.issuetype.name }}
{{end -}}
{{if .fields.assignee -}}
assignee: {{ .fields.assignee.displayName }}
{{end -}}
reporter: {{ if .fields.reporter }}{{ .fields.reporter.displayName }}{{end}}
{{if .fields.customfield_10110 -}}
watchers: {{ range .fields.customfield_10110 }}{{ .displayName }} {{end}}
{{end -}}
{{if .fields.issuelinks -}}
blockers: {{ range .fields.issuelinks }}{{if .outwardIssue}}{{ .outwardIssue.key }}[{{.outwardIssue.fields.status.name}}]{{end}}{{end}}
depends: {{ range .fields.issuelinks }}{{if .inwardIssue}}{{ .inwardIssue.key }}[{{.inwardIssue.fields.status.name}}]{{end}}{{end}}
{{end -}}
{{if .fields.priority -}}
priority: {{ .fields.priority.name }}
{{end -}}
{{if .fields.votes -}}
votes: {{ .fields.votes.votes}}
{{end -}}
{{if .fields.labels -}}
labels: {{ join ", " .fields.labels }}
{{end -}}
description: |
  {{ or .fields.description "" | indent 2 }}
{{if .fields.comment.comments}}
comments:
{{ range .fields.comment.comments }}  - | # {{.author.displayName}}, {{.created | age}} ago
    {{ or .body "" | indent 4}}
{{end}}
{{end -}}
`
const defaultEditTemplate = `{{/* edit template */ -}}
# issue: {{ .key }} - created: {{ .fields.created | age}} ago
update:
  comment:
    - add:
        body: |~
          {{ or .overrides.comment "" | indent 10 }}
fields:
  summary: >-
    {{ or .overrides.summary .fields.summary }}
{{- if and .meta.fields.components .meta.fields.components.allowedValues }}
  components: # Values: {{ range .meta.fields.components.allowedValues }}{{.name}}, {{end}}{{if .overrides.components }}{{ range (split "," .overrides.components)}}
    - name: {{.}}{{end}}{{else}}{{ range .fields.components }}
    - name: {{ .name }}{{end}}{{end}}{{end}}
{{- if .meta.fields.assignee }}
  {{- if .overrides.assignee }}
  assignee:
    emailAddress: {{ .overrides.assignee }}
  {{- else if .fields.assignee }}
  assignee: {{if .fields.assignee.name}}
    emailAddress: {{ or .fields.assignee.name}}
  {{- else }}
    emailAddress: {{.fields.assignee.emailAddress}}{{end}}{{end}}{{end}}
{{- if .meta.fields.reporter}}
  reporter:
    emailAddress: {{ if .overrides.reporter }}{{ .overrides.reporter }}{{else if .fields.reporter}}{{ .fields.reporter.emailAddress }}{{end}}{{end}}
{{- if .meta.fields.customfield_10110}}
  # watchers
  customfield_10110: {{ range .fields.customfield_10110 }}
    - name: {{ .name }}{{end}}{{if .overrides.watcher}}
    - name: {{ .overrides.watcher}}{{end}}{{end}}
{{- if .meta.fields.priority }}
  priority: # Values: {{ range .meta.fields.priority.allowedValues }}{{.name}}, {{end}}
    name: {{ or .overrides.priority .fields.priority.name "" }}{{end}}
  description: |~
    {{ or .overrides.description .fields.description "" | indent 4 }}
# votes: {{ .fields.votes.votes }}
# comments:
# {{ range .fields.comment.comments }}  - | # {{.author.displayName}}, {{.created | age}} ago
#     {{ or .body "" | indent 4 | comment}}
# {{end}}
`
const defaultTransitionsTemplate = `{{ range .transitions }}{{.id }}: {{.name}}
{{end}}`

const defaultComponentsTemplate = `{{ range . }}{{.id }}: {{.name}}
{{end}}`

const defaultComponentAddTemplate = `{{/* compoinent add template */ -}}
project: {{or .project ""}}
name: {{or .name ""}}
description: {{or .description ""}}
leadUserName: {{or .leadUserName ""}}
`

const defaultIssuetypesTemplate = `{{/* issuetypes template */ -}}
{{ range .issuetypes }}{{color "+bh"}}{{.name | append ":" | printf "%-13s" }}{{color "reset"}} {{.description}}
{{end}}`

const defaultCreateTemplate = `{{/* create template */ -}}
fields:
  project:
    key: {{ or .overrides.project "" }}
  issuetype:
    name: {{ or .overrides.issuetype "" }}
  summary: >-
    {{ or .overrides.summary "" }}{{if .meta.fields.priority.allowedValues}}
  priority: # Values: {{ range .meta.fields.priority.allowedValues }}{{.name}}, {{end}}
    name: {{ or .overrides.priority ""}}{{end}}{{if .meta.fields.components.allowedValues}}
  components: # Values: {{ range .meta.fields.components.allowedValues }}{{.name}}, {{end}}{{ range split "," (or .overrides.components "")}}
    - name: {{ . }}{{end}}{{end}}
  description: |~
    {{ or .overrides.description "" | indent 4 }}{{if .meta.fields.assignee}}
  assignee:
    emailAddress: {{ or .overrides.assignee "" }}{{end}}{{if .meta.fields.reporter}}
  reporter:
    emailAddress: {{ or .overrides.reporter .overrides.login }}{{end}}{{if .meta.fields.customfield_10110}}
  # watchers
  customfield_10110: {{ range split "," (or .overrides.watchers "")}}
    - name: {{.}}{{end}}
    - name:{{end}}`

const defaultEpicCreateTemplate = `{{/* epic create template */ -}}
fields:
  project:
    key: {{ or .overrides.project "" }}
  # Epic Name
  customfield_10120: {{ or (index .overrides "epic-name") "" }}
  summary: >-
    {{ or .overrides.summary "" }}{{if .meta.fields.priority.allowedValues}}
  priority: # Values: {{ range .meta.fields.priority.allowedValues }}{{.name}}, {{end}}
    name: {{ or .overrides.priority ""}}{{end}}{{if .meta.fields.components.allowedValues}}
  components: # Values: {{ range .meta.fields.components.allowedValues }}{{.name}}, {{end}}{{ range split "," (or .overrides.components "")}}
    - name: {{ . }}{{end}}{{end}}
  description: |~
    {{ or .overrides.description "" | indent 4 }}{{if .meta.fields.assignee}}
  assignee:
    emailAddress: {{ or .overrides.assignee "" }}{{end}}{{if .meta.fields.reporter}}
  reporter:
    emailAddress: {{ or .overrides.reporter .overrides.login }}{{end}}{{if .meta.fields.customfield_10110}}
  # watchers
  customfield_10110: {{ range split "," (or .overrides.watchers "")}}
    - name: {{.}}{{end}}
    - name:{{end}}
  issuetype:
    name: Epic`

const defaultSubtaskTemplate = `{{/* create subtask template */ -}}
fields:
  project:
    key: {{ .parent.fields.project.key }}
  summary: >-
    {{ or .overrides.summary "" }}{{if .meta.fields.priority.allowedValues}}
  priority: # Values: {{ range .meta.fields.priority.allowedValues }}{{.name}}, {{end}}
    name: {{ or .overrides.priority ""}}{{end}}{{if .meta.fields.components.allowedValues}}
  components: # Values: {{ range .meta.fields.components.allowedValues }}{{.name}}, {{end}}{{ range split "," (or .overrides.components "")}}
    - name: {{ . }}{{end}}{{end}}
  description: |~
    {{ or .overrides.description "" | indent 4 }}{{if .meta.fields.assignee}}
  assignee:
    emailAddress: {{ or .overrides.assignee "" }}{{end}}{{if .meta.fields.reporter}}
  reporter:
    emailAddress: {{ or .overrides.reporter .overrides.login }}{{end}}{{if .meta.fields.customfield_10110}}
  # watchers
  customfield_10110: {{ range split "," (or .overrides.watchers "")}}
    - name: {{.}}{{end}}
    - name:{{end}}
  issuetype:
    name: Sub-task
  parent:
    key: {{ .parent.key }}`

const defaultCommentTemplate = `body: |~
  {{ or .overrides.comment "" | indent 2 }}
`

const defaultTransitionTemplate = `{{/* transition template */ -}}
{{- if .meta.fields.comment }}
update:
  comment:
    - add:
        body: |~
          {{ or .overrides.comment "" | indent 10 }}
{{- end -}}
fields:
{{- if .meta.fields.assignee }}
  {{- if .overrides.assignee }}
  assignee:
    emailAddress: {{ .overrides.assignee }}
  {{- else if .fields.assignee }}
  assignee: {{if .fields.assignee.name}}
    emailAddress: {{ or .fields.assignee.name}}
  {{- else }}
    emailAddress: {{.fields.assignee.emailAddress}}{{end}}{{end}}
{{- end -}}
{{if .meta.fields.components}}
  components: # Values: {{ range .meta.fields.components.allowedValues }}{{.name}}, {{end}}{{if .overrides.components }}{{ range (split "," .overrides.components)}}
    - name: {{.}}{{end}}{{else}}{{ range .fields.components }}
    - name: {{ .name }}{{end}}{{end}}
{{- end -}}
{{if .meta.fields.description}}
  description: |~
    {{ or .fields.description "" | indent 4 }}
{{- end -}}
{{if .meta.fields.fixVersions -}}
  {{if .meta.fields.fixVersions.allowedValues}}
  fixVersions: # Values: {{ range .meta.fields.fixVersions.allowedValues }}{{.name}}, {{end}}{{if .overrides.fixVersions}}{{ range (split "," .overrides.fixVersions)}}
    - name: {{.}}{{end}}{{else}}{{range .fields.fixVersions}}
    - name: {{.name}}{{end}}{{end}}
  {{- end -}}
{{- end -}}
{{if .meta.fields.issuetype}}
  issuetype: # Values: {{ range .meta.fields.issuetype.allowedValues }}{{.name}}, {{end}}
    name: {{if .overrides.issuetype}}{{.overrides.issuetype}}{{else}}{{if .fields.issuetype}}{{.fields.issuetype.name}}{{end}}{{end}}
{{- end -}}
{{if .meta.fields.labels}}
  labels: {{range .fields.labels}}
    - {{.}}{{end}}{{if .overrides.labels}}{{range (split "," .overrides.labels)}}
    - {{.}}{{end}}{{end}}
{{- end -}}
{{if .meta.fields.priority}}
  priority: # Values: {{ range .meta.fields.priority.allowedValues }}{{.name}}, {{end}}
    name: {{ or .overrides.priority "unassigned" }}
{{- end -}}
{{- if .meta.fields.reporter }}
  {{- if .overrides.reporter }}
  reporter:
    name: {{ .overrides.reporter }}
  {{- else if .fields.reporter }}
  reporter: {{if .fields.reporter.name}}
    name: {{ or .fields.reporter.name}}
  {{- else }}
    displayName: {{.fields.reporter.displayName}}{{end}}{{end}}
{{- end -}}
{{if .meta.fields.resolution}}
  resolution: # Values: {{ range .meta.fields.resolution.allowedValues }}{{.name}}, {{end}}
    name: {{if .overrides.resolution}}{{.overrides.resolution}}{{else if .fields.resolution}}{{.fields.resolution.name}}{{else}}{{or .overrides.defaultResolution "Fixed"}}{{end}}
{{- end -}}
{{if .meta.fields.summary}}
  summary: >-
    {{or .overrides.summary .fields.summary}}
{{- end -}}
{{if .meta.fields.versions.allowedValues}}
  versions: # Values: {{ range .meta.fields.versions.allowedValues }}{{.name}}, {{end}}{{if .overrides.versions}}{{ range (split "," .overrides.versions)}}
    - name: {{.}}{{end}}{{else}}{{range .fields.versions}}
    - name: {{.}}{{end}}{{end}}
{{- end}}
transition:
  id: {{ .transition.id }}
  name: {{ .transition.name }}
`

const defaultWorklogTemplate = `{{/* worklog template */ -}}
# issue: {{ .issue }}
comment: |~
  {{ or .comment "" | indent 2 }}
timeSpent: {{ or .timeSpent "" }}
started: {{ or .started "" }}
`

const defaultWorklogsTemplate = `{{/* worklogs template */ -}}
{{ range .worklogs }}- # {{.author.displayName}}, {{.created | age}} ago
  comment: {{ or .comment "" }}
  started: {{ .started }}
  timeSpent: {{ .timeSpent }}

{{end}}`
