[![Build Status](https://travis-ci.org/go-jira/jira.svg?branch=master)](https://travis-ci.org/go-jira/jira)
[![GoDoc](https://godoc.org/github.com/go-jira/jira?status.svg)](https://godoc.org/github.com/go-jira/jira)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# go-jira

Simple command line client for Atlassian's Jira service written in Go.

## GDPR USERNAME DISCLAIMER

When this tool was initial written the "username" parameter was widely used in the Atlassian API.
Due to GDPR restrictions this parameter was been almost completely phased out other then V1 login.
The "--user" field is still provided as a default global, however moving forward any usage of this field should be phased out in favor of the "--login" option. 

Commands which previously took a username will now expect an email address such as watch, create, assign, etc...

## Install

### Download

You can download one of the pre-built binaries for **go-jira** [here](https://github.com/go-jira/jira/releases).

### Build

You can build and install the official repository with [Go](https://golang.org/dl/) (before running the below command, ensure you have `GO111MODULE=on` set in your environment):

	go get github.com/go-jira/jira/cmd/jira

This will checkout this repository into `$GOPATH/src/github.com/go-jira/jira/`, build, and install it.

It should then be available in $GOPATH/bin/jira.

## Usage


#### Setting up TAB completion

Since go-jira is built with the "kingpin" golang command line library we support bash/zsh shell completion automatically:

 * <https://github.com/alecthomas/kingpin/tree/v2.2.5#bashzsh-shell-completion>

For example, in bash, adding something along the lines of:

  `eval "$(jira --completion-script-bash)"`

to your bashrc, or .profile (assuming go-jira binary is already in your path) will cause jira to offer tab completion behavior.

## Configuration

**go-jira** uses a configuration hierarchy.  When loading the configuration from disk it will recursively look through all parent directories in your current path looking for a **.jira.d** directory.  If your current directory is not a child directory of your homedir, then your homedir will also be inspected for a **.jira.d** directory.  From all of **.jira.d** directories discovered **go-jira** will load a **&lt;command&gt;.yml** file (ie for `jira list` it will load `.jira.d/list.yml`) then it will merge in any properties from the **config.yml** if found.  The configuration properties found in a file closest to your current working directory will have precedence.  Properties overridden with command line options will have final precedence.

The complicated configuration hierarchy is used because **go-jira** attempts to be context aware.  For example, if you are working on a "foo" project and you `cd` into your project workspace, wouldn't it be nice if `jira ls` automatically knew to list only issues related to the "foo" project?  Likewise when you `cd` to the "bar" project then `jira ls` should only list issues related to "bar" project.  You can do this with by creating a configuration under your project workspace at **./.jira.d/config.yml** that looks like:

```yaml
project: foo
```

You will need to specify your local jira endpoint first, typically in your homedir like:

```bash
mkdir ~/.jira.d

cat <<EOM >~/.jira.d/config.yml
endpoint: https://jira.mycompany.com
EOM
```

Then use `jira login` to authenticate yourself as $USER. To change your username, use the `-u` CLI flag or set `user:` in your config.yml

### Dynamic Configuration

If the **.jira.d/config.yml** file is executable, then **go-jira** will attempt to execute the file and use the stdout for configuration.  You can use this to customize templates or other overrides depending on what type of operation you are running.  For example if you would like to use the "table" template when ever you run `jira ls`, then you can create a template like this:

```sh
#!/bin/sh

echo "endpoint: https://jira.mycompany.com"
echo "editor: emacs -nw"

case $JIRA_OPERATION in
    list)
      echo "template: table";;
esac
```

Or if you always set the same overrides when you create an issue for your project you can do something like this:

```sh
#!/bin/sh
echo "project: GOJIRA"

case $JIRA_OPERATION in
    create)
        echo "assignee: $USER"
        echo "watchers: mothra"
        ;;
esac
```

### Custom Commands
You can now create custom commands for `jira` just by editing your `.jira.d/config.yml` config file.  These commands are effectively shell-scripts that can have documented options and arguments. The basic format is like:
```yaml
custom-commands:
  - command1
  - command2
```
##### Commands
Where the individual commands are maps with these keys:
* `name: string` [**required**] This is the command name, so for `jira foobar` you would have `name: foobar`
* `help: string` This is help message displayed in the usage for the command
* `hidden: bool` This command will be hidden from users, but still executable.  Sometimes useful for constructing complex commands where one custom command might call another.
* `default: bool` Use this for compound command groups.  If you wanted to have `jira foo bar` and `jira foo baz` you would have two commands with `name: foo bar` and `name: foo baz`.  Then if you wanted `jira foo baz` to be called by default when you type `jira foo` you would set `default: true` for that custom command.
* `options: list`  This is the list of possible option flags that the command will accept
* `args: list` This is the list of command arguments (like the ISSUE) that the command will accept.
* `aliases: string list`: This is a list of alternate names that the user can provide on the command line to run the same command.  Typically used to shorten the command name or provide alternatives that users might expect.
* `script: string` [**required**] This is the script that will be executed as the action for this command. The value will be treated as a template and substitutions for options and arguments will be made before executing.

##### Options
These are possible keys under the command `options` property:
* `name: string` [**required**] Name of the option, so `name: foobar` will result in `--foobar` option.
* `help: string` The help message displayed in usage for the option.
* `type: string`:  The type of the option, can be one of these values: `BOOL`, `COUNTER`, `ENUM`, `FLOAT32`, `FLOAT64`, `INT8`, `INT16`, `INT32`, `INT64`, `INT`, `STRING`, `STRINGMAP`, `UINT8`, `UINT16`, `UINT32`,  `UINT64` and `UINT`.  Most of these are primitive data types an should be self-explanatory.  The default type is `STRING`. There are some special types:
  * `COUNTER` will be an integer type that increments each time the option is used.  So something like `--count --count` will results in `{{options.count}}` of `2`.
  * `ENUM` type is used with the `enum` property.  The raw type is a string and **must** be one of the values listed in the `enum` property.
  * `STRINGMAP` is a `string => string` map with the format of `KEY=VALUE`.  So `--override foo=bar --override bin=baz` will allow for `{{options.override.foo}}` to be `bar` and `{{options.override.bin}}` to be `baz`.
* `short: char` The single character option to be used so `short: c` will allow for `-c`.
* `required: bool` Indicate that this option must be provided on the command line.  Conflicts with the `default` property.
* `default: any` Specify the default value for the option.  Conflicts with the `required` property.
* `hidden: bool` Hide the option from the usage help message, but otherwise works fine.  Sometimes useful for developer options that user should not play with.
* `repeat: bool` Indicate that this option can be repeated.  Not applicable for `COUNTER` and `STRINGMAP` types.  This will turn the option value into an array that you can iterate over.  So `--day Monday --day Thursday` can be used like `{{range options.day}}Day: {{.}}{{end}}`
* `enum: string list` Used with the `type: ENUM` property, it is a list of strings values that represent the set of possible values the option accepts.

##### Arguments
These are possible keys under the command `args` property:
* `name: string` [**required**] Name of the option, so `name: ISSUE` will show in the usage as `jira <command> ISSUE`.  This also represents the name of the argument to be used in the script template, so `{{args.ISSUE}}`.
* `help: string` The help message displayed in usage for the argument.
* `type: string`:  The type of the argument, can be one of these values: `BOOL`, `COUNTER`, `ENUM`, `FLOAT32`, `FLOAT64`, `INT8`, `INT16`, `INT32`, `INT64`, `INT`, `STRING`, `STRINGMAP`, `UINT8`, `UINT16`, `UINT32`,  `UINT64` and `UINT`.  Most of these are primitive data types an should be self-explanatory.  The default type is `STRING`.  There are some special types:
  * `COUNTER` will be an integer type that increments each the argument is provided  So something like `jira <command> ISSUE-12 ISSUE-23` will results in `{{args.ISSUE}}` of `2`.
  * `ENUM` type is used with the `enum` property.  The raw type is a string and **must** be one of the values listed in the `enum` property.
  * `STRINGMAP` is a `string => string` map with the format of `KEY=VALUE`.  So `jira <command> foo=bar bin=baz` along with a `name: OVERRIDE` property will allow for `{{args.OVERRIDE.foo}}` to be `bar` and `{{args.OVERRIDE.bin}}` to be `baz`.
* `required: bool` Indicate that this argument must be provided on the command line.  Conflicts with the `default` property.
* `default: any` Specify the default value for the argument.  Conflicts with the `required` property.
* `repeat: bool` Indicate that this argument can be repeated.  Not applicable for `COUNTER` and `STRINGMAP` types.  This will turn the template value into an array that you can iterate over.  So `jira <command> ISSUE-12 ISSUE-23` can be used like `{{range args.ISSUE}}Issue: {{.}}{{end}}`
* `enum: string list` Used with the `type: ENUM` property, it is a list of strings values that represent the set of possible values for the argument.

##### Script Template
The `script` property is a template that would produce `/bin/sh` compatible syntax after the template has been processed.  There are 2 key template functions `{{args}}` and `{{options}}` that return the parsed arguments and option flags as a map.

To demonstrate how you might use args and options here is a `custom-test` command:
```yaml
custom-commands:
  - name: custom-test
    help: Testing the custom commands
    options:
      - name: abc
        short: a
        default: default
      - name: day
        type: ENUM
        enum:
          - Monday
          - Tuesday
          - Wednesday
          - Thursday
          - Friday
        required: true
    args:
      - name: ARG
        required: true
      - name: MORE
        repeat: true
    script: |
      echo COMMAND {{args.ARG}} --abc {{options.abc}} --day {{options.day}} {{range $more := args.MORE}}{{$more}} {{end}}
```

Then to run it:
```
$ jira custom-test
ERROR Invalid Usage: required flag --day not provided

$ jira custom-test --day Sunday
ERROR Invalid Usage: enum value must be one of Monday,Tuesday,Wednesday,Thursday,Friday, got 'Sunday'

$ jira custom-test --day Tuesday
ERROR Invalid Usage: required argument 'ARG' not provided

$ jira custom-test --day Tuesday arg1
COMMAND arg1 --abc default --day Tuesday

$ jira custom-test --day Tuesday arg1 more1 more2 more3
COMMAND arg1 --abc default --day Tuesday more1 more2 more3

$ jira custom-test --day Tuesday arg1 more1 more2 more3 --abc non-default
COMMAND arg1 --abc non-default --day Tuesday more1 more2 more3

$ jira custom-test --day Tuesday arg1 more1 more2 more3 -a short-non-default
COMMAND arg1 --abc short-non-default --day Tuesday more1 more2 more3
```

The script has access to all the environment variables that are in your current environment plus those that `jira` will set.  `jira` sets environment variables for each config property it has parsed from `.jira.d/config.yml` or the command configs at `.jira.d/<command>.yml`.  It might be useful to see all environment variables that `jira` is producing, so here is a simple custom command to list them:
```yaml
custom-commands:
  - name: env
    help: print the JIRA environment variables available to custom commands
    script: |
      env | grep JIRA
 ```

You could use the environment variables automatically, so if your `.jira.d/config.yml` looks something like this:
```yaml
project: PROJECT
custom-commands:
  - name: print-project
    help: print the name of the configured project
    script: "echo $JIRA_PROJECT"
```

##### Examples

* `jira mine` for listing issues assigned to you
```yaml
custom-commands:
  - name: mine
    help: display issues assigned to me
    script: |-
      if [ -n "$JIRA_PROJECT" ]; then
          # if `project: ...` configured just list the issues for current project
          {{jira}} list --template table --query "resolution = unresolved and assignee=currentuser() and project = $JIRA_PROJECT ORDER BY priority asc, created"
      else
          # otherwise list issues for all project
          {{jira}} list --template table --query "resolution = unresolved and assignee=currentuser() ORDER BY priority asc, created"
      fi
```
* `jira sprint` for listing issues in your current sprint
```yaml
custom-commands:
  - name: sprint
    help: display issues for active sprint
    script: |-
      if [ -n "$JIRA_PROJECT" ]; then
          # if `project: ...` configured just list the issues for current project
          {{jira}} list --template table --query "sprint in openSprints() and type != epic and resolution = unresolved and project=$JIRA_PROJECT ORDER BY rank asc, created"
      else
          # otherwise list issues for all project
          {{jira}} list --template table --query "sprint in openSprints() and type != epic and resolution = unresolved ORDER BY rank asc, created"
      fi
```

### Editing

When you run command like `jira edit` it will open up your favorite editor with the templatized output so you can quickly edit.  When the editor
closes **go-jira** will submit the completed form.  The order which **go-jira** attempts to determine your preferred editor is:

* **editor** property in any config.yml file
* **JIRA_EDITOR** environment variable
* **EDITOR** environment variable
* vim

To automate editing, use the `--noedit` to skip opening the editor, and pass `--override` fields as defined in `~/.jira.d/templates/edit`: 

```
jira edit GOJIRA-321 --noedit --override priority=P0
```

To edit fields not in the default template, [customize the template](https://github.com/go-jira/jira/issues/238#issuecomment-681666211).

### Templates

**go-jira** has the ability to customize most output (and editor input) via templates.  There are default templates available for all operations,
which may or may not work for your actual jira implementation.  Jira is endlessly customizable, so it is hard to provide default templates
that will work for all issue types.

When running a command like `jira edit` it will look through the current directory hierarchy trying to find a file that matches **.jira.d/templates/edit**,
if found it will use that file as the template, otherwise it will use the default **edit** template hard-coded into **go-jira**.  You can export the default
hard-coded templates with `jira export-templates` which will write them to **~/.jira.d/templates/**.

#### Writing/Editing Templates

First the basic templating functionality is defined by the Go language 'text/template' library.  The library reference documentation can be found [here](https://golang.org/pkg/text/template/), and there is a good primer document [here](https://gohugo.io/templates/go-templates/).  `go-jira` also provides a few extra helper functions to make it a bit easier to format the data, those functions are defined [here](https://github.com/go-jira/jira/blob/master/jiracli/templates.go#L64).

Knowing what data and fields are available to any given template is not obvious. The easiest approach to determine what is available is to use the `debug` template on any given operation.  For example to find out what is available to the "view" templates, you can use:
```
jira view GOJIRA-321 -t debug
```

This will print out the data in JSON format that is available to the template.  You can do this for any other operation, like "list":
```
jira list -t debug
```

### Authentication

#### Atlassian Cloud

For Atlassian Cloud hosted Jira [API Tokens are now required](https://developer.atlassian.com/cloud/jira/platform/deprecation-notice-basic-auth-and-cookie-based-auth/).  You will automatically be prompted for an API Token if your jira endpoint ends in `.atlassian.net`.

##### Quickstart API Token and Keychain

1. Edit your config or execute the snippit (make sure to replace `<SUBDOMAIN>` and `<EMAIL>`)
```
export SUBDOMAIN="https://<SUBDOMAIN>.atlassian.net"
export EMAIL="<EMAIL>"
mkdir -p ~/.jira.d
printf "endpoint: $SUBDOMAIN\nuser: $EMAIL\npassword-source: keyring" > ~/.jira.d/config.yml
```
2. Create a new API Token at [id.atlassian.com](https://id.atlassian.com/manage-profile/security)
3. Execute `jira session` and enter your API Token. `jira` will add your session to the keyring.

#### Private Jira Service
If you are using a private Jira service, you can force `jira` to use an api-token by setting the `authentication-method: api-token` property in your `$HOME/.jira.d/config.yml` file.  The API Token needs to be presented to the Jira service on every request, so it is recommended to store this API Token security within your OS's keyring, or using the `pass`/`gopass` service as documented below so that it can be programmatically accessed via `jira` and not prompt you every time.  For a less-secure option you can also provide the API token via a `JIRA_API_TOKEN` environment variable.  If you are unable to use an api-token for an Atlassian Cloud hosted Jira then you can still force `jira` to use the old session based authentication (until it the hosted system stops accepting it) by setting `authentication-method: session`.

The API Token authentication requires both the token and the email of the user. The email mut be set in the  `user:` in your config.yml. Failure to provide the `user` will result in a 401 error.

If your Jira service still allows you to use the Session based authentication method then `jira` will prompt for a password automatically when get a response header from the Jira service that indicates you do not have an active session (ie the `X-Ausername` header is set to `anonymous`).  Then after authentication we cache the `cloud.session.token` cookie returned by the service [session login api](https://docs.atlassian.com/jira/REST/cloud/#auth/1/session-login) and reuse that on subsequent requests.  Typically this cookie will be valid for several hours (depending on the service configuration).  To automatically securely store your password for easy reuse by jira You can enable a `password-source` via `.jira.d/config.yml` with possible values of `keyring`, `pass` or `gopass`.

#### User vs Login
The Jira service has sometimes differing opinions about how a user is identified.  In other words the ID you login with might not be ID that the jira system recognized you as.  This matters when trying to identify a user via various Jira REST APIs (like issue assignment).  This is especially relevant when trying to authenticate with an API Token where the authentication user is usually an email address, but within the Jira system the user is identified by a user name.  To accommodate this `jira` now supports two different properties in the config file.  So when authentication using the API Tokens you will likely want something like this in your `$HOME/.jira.d/config.yml` file:
```yaml
user: person
login: person@example.com
```

You can also override these values on the command line with `jira --user person --login person@example.com`.  The `login` value will be used only for authentication purposes, the `user` value will be used when a user name is required for any Jira service API calls.

#### `keyring` password source
On OSX and Linux there are a few keyring providers that `go-jira` can use (via this [golang module](https://github.com/tmc/keyring)).  To integrate `go-jira` with a supported keyring just add this configuration to `$HOME/.jira.d/config.yml`:
```yaml
password-source: keyring
```
After setting this and issuing a `jira login`, your credentials will be stored in your platform's backend (e.g. Keychain for Mac OS X) automatically. Subsequent operations, like a `jira ls`, should automatically login.

#### `pass` password source
An alternative to the keyring password source is the `pass` tool (documentation [here](https://www.passwordstore.org/)).  This uses gpg to encrypt/decrypt passwords on demand and by using `gpg-agent` you can cache the gpg credentials for a period of time so you will not be prompted repeatedly for decrypting the passwords.  The advantage over the keyring integration is that `pass` can be used on more platforms than OSX and Linux, although it does require more setup.  To use `pass` for password storage and retrieval via `go-jira` just add this configuration to `$HOME/.jira.d/config.yml`:
```yaml
password-source: pass
password-name: jira.example.com/myuser
```

This assumes you have already setup `pass` correctly on your system.  Specifically you will need to have created a gpg key like this:

```
$ gpg --gen-key
```

Then you will need the GPG Key ID you want associated with `pass`.  First list the available keys:
```
$ gpg --list-keys
/home/gojira/.gnupg/pubring.gpg
-------------------------------------------------
pub   2048R/A307D709 2016-12-18
uid                  Go Jira <gojira@example.com>
sub   2048R/F9A047B8 2016-12-18
```

Then initialize the `pass` tool to use the correct key:
```
$ pass init "Go Jira <gojira@example.com>"
```

Now insert your password with the name you configured.

```
$ pass insert jira.example.com/myuser
```

You probably want to setup gpg-agent so that you don't have to type in your gpg passphrase all the time.  You can get `gpg-agent` to automatically start by adding something like this to your `$HOME/.bashrc`
```bash
if [ -f $HOME/.gpg-agent-info ]; then
    . $HOME/.gpg-agent-info
    export GPG_AGENT_INFO
fi

if [ ! -f $HOME/.gpg-agent.conf ]; then
  cat <<EOM >$HOME/.gpg-agent.conf
default-cache-ttl 604800
max-cache-ttl 604800
default-cache-ttl-ssh 604800
max-cache-ttl-ssh 604800
EOM
fi

if [ -n "${GPG_AGENT_INFO}" ]; then
    nc  -U "${GPG_AGENT_INFO%%:*}" >/dev/null </dev/null
    if [ ! -S "${GPG_AGENT_INFO%%:*}" -o $? != 0 ]; then
        # set passphrase cache so I only have to type my passphrase once a day
        eval $(gpg-agent --options $HOME/.gpg-agent.conf --daemon --write-env-file $HOME/.gpg-agent-info --use-standard-socket --log-file $HOME/tmp/gpg-agent.log --verbose)
    fi
fi
export GPG_TTY=$(tty)
```

#### `gopass` password source
There is also the possibility to use [gopass](https://www.gopass.pw/) as a password source. `gopass` (like `pass`) uses gpg to encrypt/decrypt passwords. To use `gopass` for password storagte and retrieval via `go-jira` just add this configuration to `$HOME/.jira.d/config.yml`:
```yaml
password-source: gopass
password-name: jira.example.com/myuser
```

For this to work, you need a working `gopass` installation. 

To configure your `gpg-agent` to cache your gpg passphrase take a look at the `pass` section of the readme. 

#### `stdin` password source

When `password-source` is set to `stdin`, the `jira login` command will read from stdin until EOF, and the bytes read will be the used as the password. This is useful if you have some other programmatic method for fetching passwords. For example, if `password-generator` creates a one-time password and prints it to stdout, you could use it like this.

```bash
$ ./password-generator | jira login --endpoint=https://my.jira.endpoint.com --user=USERNAME
```

#### Switch path  used for password source
For `gopass` and `pass` it is possible to specify the full path for the `password-source` tool  used for retrieval of the password. This can be accomplised
by setting the `password-source-path` option in the configuration file. 

E.g.
```yaml
password-source: gopass
password-name: jira.example.com/myuser
password-source-path: /path/to/my-special-gopass
```

This will cause go-jira to use the `gopass` style cli interaction with the `my-special-gopass` binary.

If you ommit the `password-source-path` option, either `gopass` (for `gopass`) or `pass` (for `pass`) 
will be used.
