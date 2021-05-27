package jiracmd

import (
	"fmt"
	"net/http"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/mgutz/ansi"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func CmdLoginRegistry() *jiracli.CommandRegistryEntry {
	opts := jiracli.CommonOptions{}
	return &jiracli.CommandRegistryEntry{
		"Attempt to login into jira server",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return nil
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdLogin(o, globals, &opts)
		},
	}
}

func authCallback(req *http.Request, resp *http.Response) (*http.Response, error) {
	if resp.StatusCode == 403 {
		defer resp.Body.Close()
		// X-Authentication-Denied-Reason: CAPTCHA_CHALLENGE; login-url=https://jira/login.jsp
		if reason := resp.Header.Get("X-Authentication-Denied-Reason"); reason != "" {
			return resp, fmt.Errorf("Authenticaion Failed: " + reason)
		}
		return resp, fmt.Errorf("Authenticaion Failed: Unkown Reason")
	} else if resp.StatusCode == 200 {
		if reason := resp.Header.Get("X-Seraph-Loginreason"); reason == "AUTHENTICATION_DENIED" {
			defer resp.Body.Close()
			return resp, fmt.Errorf("Authentication Failed: " + reason)
		}
	}

	return resp, nil
}

// CmdLogin will attempt to login into jira server
func CmdLogin(o *oreo.Client, globals *jiracli.GlobalOptions, opts *jiracli.CommonOptions) error {
	if globals.AuthMethod() == "api-token" {
		log.Noticef("No need to login when using api-token authentication method")
		return nil
	}

	ua := o.WithoutRedirect().WithRetries(0).WithoutCallbacks().WithPostCallback(authCallback)

	if session, err := jira.GetSession(o, globals.Endpoint.Value); err != nil {
		// No active session so try to create a new one
		_, err := jira.NewSession(ua, globals.Endpoint.Value, globals)
		if err != nil {
			// reset password on failed session
			globals.SetPass("")
			log.Errorf("%s", err)
		} else if !globals.Quiet.Value {
			fmt.Println(ansi.Color("OK", "green"), "New session for", globals.User)
		}
	} else {
		if !globals.Quiet.Value {
			fmt.Println(ansi.Color("OK", "green"), "Found session for", session.Name)
		}
	}
	return nil
}
