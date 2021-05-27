package jira

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/coryb/oreo"
	"github.com/go-jira/jira/jiradata"
)

// https://docs.atlassian.com/jira/REST/cloud/#api/2/project-getProjectComponents
func (j *Jira) GetProjectComponents(project string) (*jiradata.Components, error) {
	return GetProjectComponents(j.UA, j.Endpoint, project)
}

func GetProjectComponents(ua HttpClient, endpoint string, project string) (*jiradata.Components, error) {
	uri := URLJoin(endpoint, "rest/api/2/project", project, "components")
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := jiradata.Components{}
		return &results, json.NewDecoder(resp.Body).Decode(&results)
	}
	return nil, responseError(resp)
}

// https://developer.atlassian.com/cloud/jira/platform/rest/v2#api-api-2-project-projectIdOrKey-versions-get
func (j *Jira) GetProjectVersions(project string) (*jiradata.Versions, error) {
	return GetProjectVersions(j.UA, j.Endpoint, project)
}

func GetProjectVersions(ua HttpClient, endpoint string, project string) (*jiradata.Versions, error) {
	uri := URLJoin(endpoint, "rest/api/2/project", project, "versions")
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		results := jiradata.Versions{}
		return &results, json.NewDecoder(resp.Body).Decode(&results)
	}
	return nil, responseError(resp)
}

func GetProjectVersionsPaginated(ua HttpClient, endpoint string, project string, status []string, query string, order string) (*jiradata.Versions, error) {
	startAt := 0
	total := 1
	maxResults := 100
	releases := jiradata.Versions{}
	for startAt < total {
		uri, err := url.Parse(URLJoin(endpoint, "rest/api/2/project", project, "version"))
		if err != nil {
			return nil, err
		}

		params := url.Values{}
		if len(status) > 0 {
			params.Add("status", strings.Join(status, ","))
		}
		if len(query) > 0 {
			params.Add("query", query)
		}
		if len(order) > 0 {
			params.Add("orderBy", order)
		}
		params.Add("maxResults", fmt.Sprintf("%d", maxResults))
		params.Add("startAt", fmt.Sprintf("%d", startAt))

		uri.RawQuery = params.Encode()

		resp, err := ua.Do(oreo.RequestBuilder(uri).WithHeader("Accept", "application/json").Build())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			results := &jiradata.PageOfVersion{}
			err := json.NewDecoder(resp.Body).Decode(&results)
			if err != nil {
				return nil, err
			}
			startAt = startAt + maxResults
			total = results.Total
			releases = append(releases, results.Values...)
		} else {
			return nil, responseError(resp)
		}
	}
	return &releases, nil
}
