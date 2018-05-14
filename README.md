# cu-sts
cu-sts interacts with Cornell's existing Shibboleth+DUO AWS integration to create temporary STS credentials from your existing `shib-*` groups. This allows you to avoid making IAM users and permanent keys while still using existing AD group membership.

# Requirements
- Chrome >= 59 (cu-sts uses Headless Chrome in the background)

# Installation
Download the [latest release](https://github.com/ian-d/cu-sts/releases). Releases are available for OS X, Linux, and Windows (experimental).

# Usage
cu-sts has two authentication modes: ad-hoc and config file. Either can be used with any cu-sts command.

## Ad-Hoc
Ad-hoc usage requires, at minimum, these flags:
- `--username`, the NetID to use for Shibboleth login
- `--account`, the twelve digit AWS account ID
- `--role`, the IAM role with SAML federated access enabled to generate STS credentials for (`shib-admin`, `shib-dev`, etc)

## Config File
cu-sts by default will look for the config file `~/.cu-sts.toml`. A config file can contain defaults for common flags and a list of pre-defined "profiles":
```
username = "isd23"
duo_method = "push"

[profile.admin]
account = "0123456789"
role = "shib-admin"
duration = 900

[profile.dev]
account = "0123456789"
role = "shib-dev"
```

Profiles can be reference by name via the `--profile` or `--profiles` flag.

# Commands
cu-sts has two main commands: `exec` and `creds`, both of with can use either ad-hoc or config file profiles.

## exec
`cu-sts exec` will generate STS credentials and run a command in sub-shell with the appropriate `AWS_*` environment variables:
```
cu-sts exec --profile=admin -- aws sts get-caller-identity
Loaded config file: /Users/isd23/.cu-sts.toml
Password: ************
(chrome) Fetching IdP Shibboleth login page.
(chrome) Submitting username & password.
(chrome) Submitting selected DUO method.
(chrome) Auto-selected DUO method used, ignoring configured method 'push'.
(chrome) Waiting for DUO response and SAML assertion.
Received AWS STS credentials for admin, spawning sub-command.
{
    "Account": "225162606092",
    "UserId": "AROAINI5MJRGSU6QABXHK:isd23@cornell.edu",
    "Arn": "arn:aws:sts::225162606092:assumed-role/shib-cli/isd23@cornell.edu"
}
```

If a sub-command is not included then `exec` will simply spawn `$SHELL` and will also set `CUSTS_PROFILE` in the environment to the profile or account/role used to generate the current credentials (useful for custom shell prompt):
```
➜  ~ cu-sts exec --profile=admin
Loaded config file: /Users/isd23/.cu-sts.toml
Password: ************
...
Received AWS STS credentials for admin, spawning sub-command.
[admin]➜  ~
```

## creds
`creds` generates credentials and saves them an external file (default `~/.aws/credentials`). This is useful if you're used to working with `AWS_PROFILE` set or using the `--profile` flag in the AWS CLI. Multiple config file profiles can be used at one time:
```
$ cu-sts creds --profiles=admin,dev
Loaded config file: /Users/isd23/.cu-sts.toml
Password: ************
...
Writing credentials to /Users/isd23/.aws/credentials.
Received AWS STS credentials for admin, writing to file.
Received AWS STS credentials for dev, writing to file.

$ grep -A1 -E "(admin|dev)" ~/.aws/credentials
[admin]
aws_access_key_id     = ASIAI7AXKP6Z5ESQNFSQ
--
[dev]
aws_access_key_id     = ASIAI5KI2WNIW2FH6JIQ
```

## Known Issues
[chromedp](https://github.com/chromedp/chromedp) has an outstanding bug that can cause a ~7s hang while waiting for all DOM events to complete before an element is considered "ready": ["domEvent: timeout waiting for node"](https://github.com/chromedp/chromedp/issues/75)

cu-sts uses a vendored version with suppressed error messages, since it's still functional, just annoying: vendor/chromedp/chromedp/handler.go#646
