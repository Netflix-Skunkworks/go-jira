export COLUMNS=149
if command -v stty 1>/dev/null 2>/dev/null
then
  stty columns 149
fi
export JIRA_LOG_FORMAT="%{level:-5s} %{message}"
export ENDPOINT="https://go-jira.atlassian.net"
export GNUPGHOME=$(pwd)/.gnupg
export PASSWORD_STORE_DIR=$(pwd)/.password-store
export JIRACLOUD=1

