package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/coryb/oreo"

	"github.com/go-jira/jira/jiradata"
)

type IssueQueryProvider interface {
	ProvideIssueQueryString() string
}

type IssueOptions struct {
	Fields        []string `json:"fields,omitempty" yaml:"fields,omitempty"`
	Expand        []string `json:"expand,omitempty" yaml:"expand,omitempty"`
	Properties    []string `json:"properties,omitempty" yaml:"properties,omitempty"`
	FieldsByKeys  bool     `json:"fieldsByKeys,omitempty" yaml:"fieldsByKeys,omitempty"`
	UpdateHistory bool     `json:"updateHistory,omitempty" yaml:"updateHistory,omitempty"`
}

func (o *IssueOptions) ProvideIssueQueryString() string {
	params := []string{}
	if len(o.Fields) > 0 {
		params = append(params, "fields="+strings.Join(o.Fields, ","))
	}
	if len(o.Expand) > 0 {
		params = append(params, "expand="+strings.Join(o.Expand, ","))
	}
	if len(o.Properties) > 0 {
		params = append(params, "properties="+strings.Join(o.Properties, ","))
	}
	if o.FieldsByKeys {
		params = append(params, "fieldsByKeys=true")
	}
	if o.UpdateHistory {
		params = append(params, "updateHistory=true")
	}
	if len(params) > 0 {
		return "?" + strings.Join(params, "&")
	}
	return ""
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-getIssue
func (j *Jira) GetIssue(issue string, iqg IssueQueryProvider) (*jiradata.Issue, error) {
	return GetIssue(j.UA, j.Endpoint, issue, iqg)
}

func GetIssue(ua HttpClient, endpoint string, issue string, iqg IssueQueryProvider) (*jiradata.Issue, error) {
	query := ""
	if iqg != nil {
		query = iqg.ProvideIssueQueryString()
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue)
	uri += query
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := &jiradata.Issue{}
		return results, json.NewDecoder(resp.Body).Decode(results)
	}
	return nil, responseError(resp)
}

func (j *Jira) GetIssueWorklog(issue string) (*jiradata.Worklogs, error) {
	return GetIssueWorklog(j.UA, j.Endpoint, issue)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue/{issueIdOrKey}/worklog-getIssueWorklog
func GetIssueWorklog(ua HttpClient, endpoint string, issue string) (*jiradata.Worklogs, error) {
	startAt := 0
	total := 1
	maxResults := 100
	worklogs := jiradata.Worklogs{}
	for startAt < total {
		uri := URLJoin(endpoint, "rest/api/2/issue", issue, "worklog")
		uri += fmt.Sprintf("?startAt=%d&maxResults=%d", startAt, maxResults)
		resp, err := ua.GetJSON(uri)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			results := &jiradata.WorklogWithPagination{}
			err := json.NewDecoder(resp.Body).Decode(results)
			if err != nil {
				return nil, err
			}
			startAt = startAt + maxResults
			total = results.Total
			worklogs = append(worklogs, results.Worklogs...)
		} else {
			return nil, responseError(resp)
		}
	}
	return &worklogs, nil
}

func (j *Jira) GetIssueComment(issue string) (*jiradata.Comments, error) {
	return GetIssueComment(j.UA, j.Endpoint, issue)
}

// https://docs.atlassian.com/software/jira/docs/api/REST/7.12.0/#api/2/issue-getComments
func GetIssueComment(ua HttpClient, endpoint string, issue string) (*jiradata.Comments, error) {
	startAt := 0
	total := 1
	maxResults := 100
	comments := jiradata.Comments{}
	for startAt < total {
		uri := URLJoin(endpoint, "rest/api/2/issue", issue, "comment")
		uri += fmt.Sprintf("?startAt=%d&maxResults=%d", startAt, maxResults)
		resp, err := ua.GetJSON(uri)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			results := &jiradata.CommentsWithPagination{}
			err := json.NewDecoder(resp.Body).Decode(results)
			if err != nil {
				return nil, err
			}
			startAt = startAt + maxResults
			total = results.Total
			comments = append(comments, results.Comments...)
		} else {
			return nil, responseError(resp)
		}
	}
	return &comments, nil
}

type WorklogProvider interface {
	ProvideWorklog() *jiradata.Worklog
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue/{issueIdOrKey}/worklog-addWorklog
func (j *Jira) AddIssueWorklog(issue string, wp WorklogProvider) (*jiradata.Worklog, error) {
	return AddIssueWorklog(j.UA, j.Endpoint, issue, wp)
}

func AddIssueWorklog(ua HttpClient, endpoint string, issue string, wp WorklogProvider) (*jiradata.Worklog, error) {
	req := wp.ProvideWorklog()
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "worklog")
	resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		results := &jiradata.Worklog{}
		return results, json.NewDecoder(resp.Body).Decode(results)
	}
	return nil, responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-getEditIssueMeta
func (j *Jira) GetIssueEditMeta(issue string) (*jiradata.EditMeta, error) {
	return GetIssueEditMeta(j.UA, j.Endpoint, issue)
}

func GetIssueEditMeta(ua HttpClient, endpoint string, issue string) (*jiradata.EditMeta, error) {
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "editmeta")
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := &jiradata.EditMeta{}
		return results, json.NewDecoder(resp.Body).Decode(results)
	}
	return nil, responseError(resp)
}

type IssueUpdateProvider interface {
	ProvideIssueUpdate() *jiradata.IssueUpdate
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-editIssue
func (j *Jira) EditIssue(issue string, iup IssueUpdateProvider) error {
	return EditIssue(j.UA, j.Endpoint, issue, iup)
}

func EditIssue(ua HttpClient, endpoint string, issue string, iup IssueUpdateProvider) error {
	req := iup.ProvideIssueUpdate()
	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue)
	resp, err := ua.Put(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-createIssue
func (j *Jira) CreateIssue(iup IssueUpdateProvider) (*jiradata.IssueCreateResponse, error) {
	return CreateIssue(j.UA, j.Endpoint, iup)
}

func CreateIssue(ua HttpClient, endpoint string, iup IssueUpdateProvider) (*jiradata.IssueCreateResponse, error) {
	req := iup.ProvideIssueUpdate()
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue")
	resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		results := &jiradata.IssueCreateResponse{}
		return results, json.NewDecoder(resp.Body).Decode(results)
	}
	return nil, responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-getCreateIssueMeta
func (j *Jira) GetIssueCreateMetaProject(projectKey string) (*jiradata.CreateMetaProject, error) {
	return GetIssueCreateMetaProject(j.UA, j.Endpoint, projectKey)
}

func GetIssueCreateMetaProject(ua HttpClient, endpoint string, projectKey string) (*jiradata.CreateMetaProject, error) {
	uri := URLJoin(endpoint, "rest/api/2/issue/createmeta")
	uri += fmt.Sprintf("?projectKeys=%s&expand=projects.issuetypes.fields", projectKey)
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := &jiradata.CreateMeta{}
		err = json.NewDecoder(resp.Body).Decode(results)
		if err != nil {
			return nil, err
		}
		for _, project := range results.Projects {
			if project.Key == projectKey {
				return project, nil
			}
		}
		return nil, fmt.Errorf("project %s not found", projectKey)
	}
	return nil, responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-getCreateIssueMeta
func (j *Jira) GetIssueCreateMetaIssueType(projectKey, issueTypeName string) (*jiradata.IssueType, error) {
	return GetIssueCreateMetaIssueType(j.UA, j.Endpoint, projectKey, issueTypeName)
}

func GetIssueCreateMetaIssueType(ua HttpClient, endpoint string, projectKey, issueTypeName string) (*jiradata.IssueType, error) {
	uri := URLJoin(endpoint, "rest/api/2/issue/createmeta")
	uri += fmt.Sprintf("?projectKeys=%s&issuetypeNames=%s&expand=projects.issuetypes.fields", projectKey, url.QueryEscape(issueTypeName))
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, responseError(resp)
	}
	results := &jiradata.CreateMeta{}
	if err := json.NewDecoder(resp.Body).Decode(results); err != nil {
		return nil, err
	}
	for _, project := range results.Projects {
		if project.Key != projectKey {
			continue
		}
		for _, issueType := range project.IssueTypes {
			if issueType.Name == issueTypeName {
				return issueType, nil
			}
		}
	}
	return nil, fmt.Errorf("project %s and IssueType %s not found", projectKey, issueTypeName)
}

type LinkIssueProvider interface {
	ProvideLinkIssueRequest() *jiradata.LinkIssueRequest
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issueLink-linkIssues
func (j *Jira) LinkIssues(lip LinkIssueProvider) error {
	return LinkIssues(j.UA, j.Endpoint, lip)
}

func LinkIssues(ua HttpClient, endpoint string, lip LinkIssueProvider) error {
	req := lip.ProvideLinkIssueRequest()
	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}
	uri := URLJoin(endpoint, "rest/api/2/issueLink")
	resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		return nil
	}
	return responseError(resp)
}

// https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issue-remote-links/#api-rest-api-2-issue-issueidorkey-remotelink-post
func (j *Jira) LinkRemoteIssue(issue string, url string, title string) error {
	return LinkRemoteIssue(j.UA, j.Endpoint, issue, url, title)
}

func LinkRemoteIssue(ua HttpClient, endpoint string, issue string, url string, title string) error {
        req := struct {
            Object struct {
                Url   string `json:"url"`
                Title string `json:"title"`
            } `json:"object"`
        }{}
        req.Object.Url = url
        req.Object.Title = title

	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}

	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "remotelink")
	resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		return nil
	}
	return responseError(resp)
}


// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-getTransitions
func (j *Jira) GetIssueTransitions(issue string) (*jiradata.TransitionsMeta, error) {
	return GetIssueTransitions(j.UA, j.Endpoint, issue)
}

func GetIssueTransitions(ua HttpClient, endpoint string, issue string) (*jiradata.TransitionsMeta, error) {
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "transitions")
	uri += "?expand=transitions.fields"
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := &jiradata.TransitionsMeta{}
		return results, json.NewDecoder(resp.Body).Decode(results)
	}
	return nil, responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-doTransition
func (j *Jira) TransitionIssue(issue string, iup IssueUpdateProvider) error {
	return TransitionIssue(j.UA, j.Endpoint, issue, iup)
}

func TransitionIssue(ua HttpClient, endpoint string, issue string, iup IssueUpdateProvider) error {
	req := iup.ProvideIssueUpdate()
	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "transitions")
	resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issueLinkType-getIssueLinkTypes
func (j *Jira) GetIssueLinkTypes() (*jiradata.IssueLinkTypes, error) {
	return GetIssueLinkTypes(j.UA, j.Endpoint)
}

func GetIssueLinkTypes(ua HttpClient, endpoint string) (*jiradata.IssueLinkTypes, error) {
	uri := URLJoin(endpoint, "rest/api/2/issueLinkType")
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := struct {
			IssueLinkTypes jiradata.IssueLinkTypes
		}{
			IssueLinkTypes: jiradata.IssueLinkTypes{},
		}
		return &results.IssueLinkTypes, json.NewDecoder(resp.Body).Decode(&results)
	}
	return nil, responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-addVote
func (j *Jira) IssueAddVote(issue string) error {
	return IssueAddVote(j.UA, j.Endpoint, issue)
}

func IssueAddVote(ua HttpClient, endpoint string, issue string) error {
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "votes")
	resp, err := ua.Post(uri, "application/json", strings.NewReader("{}"))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-removeVote
func (j *Jira) IssueRemoveVote(issue string) error {
	return IssueRemoveVote(j.UA, j.Endpoint, issue)
}

func IssueRemoveVote(ua HttpClient, endpoint string, issue string) error {
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "votes")
	resp, err := ua.Delete(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

type RankRequestProvider interface {
	ProvideRankRequest() *jiradata.RankRequest
}

// https://docs.atlassian.com/jira-software/REST/cloud/#agile/1.0/issue-rankIssues
func (j *Jira) RankIssues(rrp RankRequestProvider) error {
	return RankIssues(j.UA, j.Endpoint, rrp)
}

func RankIssues(ua HttpClient, endpoint string, rrp RankRequestProvider) error {
	req := rrp.ProvideRankRequest()
	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}
	uri := URLJoin(endpoint, "rest/agile/1.0/issue/rank")
	resp, err := ua.Put(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-addWatcher
func (j *Jira) IssueAddWatcher(issue, user string) error {
	return IssueAddWatcher(j.UA, j.Endpoint, issue, user)
}

func IssueAddWatcher(ua HttpClient, endpoint string, issue, user string) error {
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "watchers")
	resp, err := ua.Post(uri, "application/json", strings.NewReader(fmt.Sprintf("%q", user)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-addWatcher
func (j *Jira) IssueRemoveWatcher(issue, user string) error {
	return IssueRemoveWatcher(j.UA, j.Endpoint, issue, user)
}

func IssueRemoveWatcher(ua HttpClient, endpoint string, issue, user string) error {
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "watchers")
	uri += fmt.Sprintf("?accountId=%s", user)
	resp, err := ua.Delete(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

type CommentProvider interface {
	ProvideComment() *jiradata.Comment
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue/{issueIdOrKey}/comment-addComment
func (j *Jira) IssueAddComment(issue string, cp CommentProvider) (*jiradata.Comment, error) {
	return IssueAddComment(j.UA, j.Endpoint, issue, cp)
}

func IssueAddComment(ua HttpClient, endpoint string, issue string, cp CommentProvider) (*jiradata.Comment, error) {
	req := cp.ProvideComment()
	encoded, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "comment")
	resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		results := jiradata.Comment{}
		return &results, json.NewDecoder(resp.Body).Decode(&results)
	}
	return nil, responseError(resp)
}

type UserProvider interface {
	ProvideUser() *jiradata.User
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue-assign
func (j *Jira) IssueAssign(issue, name string) error {
	return IssueAssign(j.UA, j.Endpoint, issue, name)
}

func IssueAssign(ua HttpClient, endpoint string, issue, name string) error {
	// this is special, not using the jiradata.User structure
	// because we need to be able to send `null` as the name param
	// when we want to un-assign the issue
	req := struct {
		Name *string `json:"name"`
	}{&name}
	if name == "" {
		req.Name = nil
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "assignee")
	resp, err := ua.Put(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

func IssueAssignAccountID(ua HttpClient, endpoint string, issue, acctId string) error {
	// this is special, not using the jiradata.User structure
	// because we need to be able to send `null` as the name param
	// when we want to un-assign the issue
	req := struct {
		AccountID *string `json:"accountId"`
	}{&acctId}
	if acctId == "" {
		req.AccountID = nil
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		return err
	}
	uri := URLJoin(endpoint, "rest/api/2/issue", issue, "assignee")
	resp, err := ua.Put(uri, "application/json", bytes.NewBuffer(encoded))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	return responseError(resp)
}

// https://docs.atlassian.com/jira/REST/cloud/#api/2/issue/{issueIdOrKey}/attachments-addAttachment
func (j *Jira) IssueAttachFile(issue, filename string, contents io.Reader) (*jiradata.ListOfAttachment, error) {
	return IssueAttachFile(j.UA, j.Endpoint, issue, filename, contents)
}

func IssueAttachFile(ua HttpClient, endpoint string, issue, filename string, contents io.Reader) (*jiradata.ListOfAttachment, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	formFile, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(formFile, contents)
	if err != nil {
		return nil, err
	}

	uri, err := url.Parse(URLJoin(endpoint, "rest/api/2/issue", issue, "attachments"))
	if err != nil {
		return nil, err
	}
	req := oreo.RequestBuilder(uri).WithMethod("POST").WithHeader(
		"X-Atlassian-Token", "no-check",
	).WithHeader(
		"Accept", "application/json",
	).WithContentType(w.FormDataContentType()).WithBody(&buf).Build()
	w.Close()

	resp, err := ua.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := jiradata.ListOfAttachment{}
		return &results, json.NewDecoder(resp.Body).Decode(&results)
	}
	return nil, responseError(resp)
}
