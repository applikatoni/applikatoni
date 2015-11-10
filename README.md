![Applikatoni - Deployments Al Forno](https://s3.eu-central-1.amazonaws.com/applikatoni/assets/banner_with_slogan.png)
[![Build Status](https://travis-ci.org/applikatoni/applikatoni.svg?branch=master)](https://travis-ci.org/applikatoni/applikatoni) [![GoDoc](https://godoc.org/github.com/applikatoni/applikatoni?status.svg)](https://godoc.org/github.com/applikatoni/applikatoni)

**Applikatoni is a self-hosted deployment server for small teams, that takes
your code from GitHub and deploys it to your servers, by running shell commands
you define.**

# Introduction

Applikatoni was developed at [flinc](https://flinc.org) for internal use and
later open sourced.

With its web-frontend and its deep **GitHub** integration it allows you to
deploy your applications to multiple servers with the click of a button.

Besides giving your team **a history of deployments**, so you can see what has
been deployed by whom to which servers, it also offers integration into a
multitude of a **third-party services**, so your team never misses a deployment.
**Deploy pull requests** from GitHub, check the **Travis CI** status of a branch
before deploying, notify your team in **Flowdock** about deployments, tell
**NewRelic** about the current revision of your application and reset
**Bugsnag** after a deployment.

**No database server is needed**. Applikatoni stores information about deployment in
a sqlite3 file. Easy to use, easy to maintain and easy to backup.

At its core, **Applikatoni is technology agnostic**. Deploy Rails, Node.js, Java
and Go applications. It doesn't matter to Applikatoni. It doesn't do more than
taking your code from GitHub and deploying it to your servers by running the
shell commands you specify.

You can run different scripts on different hosts, if they fulfill different roles
in your system. Your application server may need to run different commands than
your background worker server, or your database server. Applikatoni can handle
all of this.

With the ability to deploy **multiple applications to different servers with
varying deployment strategies** you can use Applikatoni however you like.

And if clicking isn't your thing and you prefer a terminal over a web frontend:
check out [toni](https://github.com/applikatoni/toni) -- the CLI for your
Applikatoni server.

Also: there is a lot of pizza involved! 🍕

# Getting started

1. Create a GitHub Application for your organization so Applikatoni can access
   your code:

          https://github.com/organizations/<YOUR ORGANIZATION>/settings/applications

    Enter the OAuth2 callback URL:

          <where your applikatoni instance will be hosted>/oauth2/callback

2. Install Appliatoni on your server. See [Installation](#installation) for more
   information

3. Configure Applikatoni. See [Configuration](#configuration) for detailed
   instructions.

4. Setup the database with the packaged
   [goose](https://bitbucket.org/liamstask/goose) binary. See [Usage](#usage) on how to do that.

5. Start Applikatoni

# Installation
## Dependencies

* sqlite3
* goose - [https://bitbucket.org/liamstask/goose/](https://bitbucket.org/liamstask/goose/)

## Download a packaged version

See the [Releases](https://github.com/applikatoni/applikatoni/releases) section
and download a packaged version of Applikatoni.

## Installing using `go get`

        go get github.com/applikatoni/applikatoni/server
# Usage

1. Make sure the database file is setup and migrated:

        go get bitbucket.org/liamstask/goose/cmd/goose
        vim db/dbconf.yml
        goose -env="production" up
2. Create a `configuration.json` file for your needs. See [Configuration](#configuration) for more information.

        cp configuration_example.json configuration.json
        vim configuration.json
3. Start the server:

        ./applikatoni -port=:8080 -db=./db/production.db -conf=./configuration.json -env=production

# How it works

Applikatoni is a server with a web-frontend that allows users to deploy specific
commits hosted on GitHub to multiple servers.

A deployment process consists of multiple `stages`, which Applikatoni will
execute sequentially on each server, but at the same time on all servers.

A `stage` is nothing more than a few lines of shell commands, that you specify.
These shell commands can contain variables, like `CommitSha`, `RubyVersion`,
`WorkingDirectory` and so on. Before the deployment process starts, Applikatoni
"renders" these scripts templates into fully executable commands, by injecting
the values of the variables.

With the templates fully rendered, Applikatoni then connects to the specified
servers via SSH (and a username and private SSH key you specify) and executes
these shell commands.

If one stage fails, the whole deployment process is stopped after the failed
stage has finished on all servers, so there is no inconsistent state.

Which stages are executed on which servers depends on which `roles` each server
fulfills in your system. If the server `one.your-company.com` has the role
"application-server" and `two.your-company.com` has the role "queue-server" only
the stages specified for the respective stages get executed on the servers.

Here is an excerpt from an example `configuration.json`:

```json
"default_stages": ["PRE_DEPLOYMENT", "CODE_DEPLOYMENT", "POST_DEPLOYMENT"],
"available_stages": ["PRE_DEPLOYMENT", "CODE_DEPLOYMENT", "MIGRATE_DATABASE", "POST_DEPLOYMENT"],
"hosts": [
  {
    "name": "web.shipping-company.com:22",
    "roles": ["web", "migrator"]
  },
  {
    "name": "workers.shipping-company.com:22",
    "roles": ["workers"]
  }
],
"roles": [
  {
    "name": "workers",
    "script_templates": {
      "PRE_DEPLOYMENT": "sudo /etc/init.d/workers stop",
      "CODE_DEPLOYMENT": "cd {{.Dir}}/current && git fetch origin && git reset -q --hard {{.CommitSha}}",
      "POST_DEPLOYMENT": "sudo /etc/init.d/workers start"
    },
    "options": {
      "Dir": "/var/www/our-main-application"
    }
  },
  {
    "name": "web",
    "script_templates": {
      "PRE_DEPLOYMENT": "sudo /etc/init.d/application-server prepare",
      "CODE_DEPLOYMENT": "cd {{.Dir}}/current && git fetch origin && git reset -q --hard {{.CommitSha}}",
      "POST_DEPLOYMENT": "sudo /etc/init.d/application-server reload"
    },
    "options": {
      "Dir": "/var/www/our-main-application"
    }
  },
  {
    "name": "migrator",
    "script_templates": {
      "MIGRATE_DATABASE": "cd {{.Dir}} && make migrate-database"
    },
    "options": {
      "Dir": "/var/www/our-main-application",
    }
  }
]
```

The `web.shipping-company.com` host has the roles `web` and `migrator`. The
`workers.shipping-company.com` host has the role `workers`.

With a default deployment, the stages `PRE_DEPLOYMENT`, `CODE_DEPLOYMENT` and
`POST_DEPLOYMENT` will be run on those two hosts.

If you also select `MIGRATE_DATABASE` before creating the deployment, the stages
`PRE_DEPLOYMENT`, `CODE_DEPLOYMENT`, `MIGRATE_DATABASE` and `POST_DEPLOYMENT`
will be run on those two hosts. In this order. And since
`workers.shipping-company.com` doesn't fulfill the `migrator` role, the
`MIGRATE_DATABASE` stage will be skipped on this server.

Before running the commands, Applikatoni converts this:

    cd {{.Dir}}/current && git fetch origin && git reset -q --hard {{.CommitSha}}

into this:

    cd /var/www/our-main-application/current && git fetch origin && git reset -q --hard F00B4R

where `F00B4R` is the commit SHA you selected in the web frontend.

# Terminology

* `application` - Applikatoni can deploy multiple applications
* `target` - The target environment of a deployment, e.g.: `production`
* `host` - Each `target` has one or more `host`s, the servers to which to deploy
  to. e.g.: `web-application.production.shipping-company.com`
* `role` - Every `host` of each `target` fulfills different `role`s. What gets
  executed and when on which `host` depends on the `role`s this host has. e.g.:
`database-server` or `web-app-server`.
* `stage` - A deployment consists of on or more stages. A `role` defines on or
  more `stage`s (by defining `script_templates` for each `stage`). A `stage` can
  succeed or fail while deploying the application. If a `stage` failed, the
  deployment stops after this change. All `stage`s are executed synchronously on
  all `host`s (but in parallel).

# Configuration
## Requirements
* a user on the servers you want to deploy your application to with a SSH-Key
* a GitHub OAuth application ([See here](https://github.com/settings/applications/new))

## The configuration.json

The Applikatoni is configured by reading a `configuration.json` file.

You can find a simplified example for a Rails application with two Unicorn web
servers and one Sidekiq worker server here:
[configuration_example.json](./configuration_example.json). Or read on to get a
run down of what it's doing.

### Sample

Here is a sample `configuration.json` for an application called
`our-main-application` with its source code hosted under
`github.com/shipping-company/our-main-application`.

The application will be deployed to three servers:

* 1.unicorn.production.shipping-company.com
* 2.unicorn.production.shipping-company.com
* 1.workers.production.shipping-company.com

The deployment process reflects one of a typical Rails application with the
Unicorn webserver and Sidekiq as a background worker process.

1. Shut down sidekiq
2. Pull down the code from GitHub (the `CommitSha` variable is injected by
   Applikatoni into the script templates and represents the commit the user
   wants to deploy)
3. Install dependencies
4. Hot-reload Unicorn / start sidekiq

```json
{
  "ssl_enabled": false,
  "host": "applikatoni.shipping-company.com",
  "session_secret": "<SECRET>",
  "oauth2_state_string": "<UNGUESSABLE RANDOM OAUTH2 STATE STRING>",
  "github_client_id": "<CLIENT_ID>",
  "github_client_secret": "<CLIENT_SECRET>",
  "mandrill_api_key": "<API_KEY>",
  "applications": [
    {
      "name": "our-main-application",
      "read_usernames": ["<CAN READ DEPLOYMENT HISTORY BUT NOT DEPLOY>"],
      "github_owner": "shipping-company",
      "github_repo": "our-main-application",
      "github_branches": ["master", "develop", "production", "production"],
      "travis_image_url": "https://magnum.travis-ci.com/shipping-company/our-main-application.svg?token=<KEY HERE>",
      "daily_digest_receivers": ["team@shipping-company.com"],
      "daily_digest_target": "production",
      "targets": [
        {
          "name": "production",
          "deployment_user": "deploy",
          "deployment_ssh_key": "<SSH KEY>",
          "deploy_usernames": ["<YOUR GITHUB USERNAME>"],
          "default_stages": ["CHECK_CONNECTION", "PRE_DEPLOYMENT", "CODE_DEPLOYMENT", "POST_DEPLOYMENT"],
          "available_stages": ["CHECK_CONNECTION", "PRE_DEPLOYMENT", "CODE_DEPLOYMENT", "MIGRATE_DATABASE", "POST_DEPLOYMENT"],
          "bugsnag_api_key": "<YOUR BUGSNAG API KEY>",
          "flowdock_endpoint": "<FLOWDOCK REST ENTRY POINT WITH HTTP AUTH>",
          "new_relic_api_key": "<NEW RELIC API KEY>",
          "new_relic_app_id": "<NEW RELIC APP ID>",
          "hosts": [
            {
              "name": "1.unicorn.production.shipping-company.com:22",
              "roles": ["web", "migrator"]
            },
            {
              "name": "2.unicorn.production.shipping-company.com:22",
              "roles": ["web"]
            },
            {
              "name": "1.workers.production.shipping-company.com:22",
              "roles": ["workers"]
            }
          ],
          "roles": [
            {
              "name": "workers",
              "script_templates": {
                "CHECK_CONNECTION": "test -d {{.Dir}}",
                "PRE_DEPLOYMENT": "sudo /etc/init.d/sidekiq stop",
                "CODE_DEPLOYMENT": "cd {{.Dir}}/current && git fetch --tags -q origin && git reset -q --hard {{.CommitSha}}\ncd {{.Dir}}/current && RAILS_ENV={{.RailsEnv}} bundle install --gemfile {{.Dir}}/current/Gemfile --path {{.Dir}}/shared/bundle --deployment --quiet --without development test",
                "POST_DEPLOYMENT": "sudo /etc/init.d/sidekiq start"
              },
              "options": {
                "Dir": "/var/www/our-main-application",
                "RailsEnv": "production"
              }
            },
            {
              "name": "web",
              "script_templates": {
                "CHECK_CONNECTION": "test -d {{.Dir}}",
                "PRE_DEPLOYMENT": "echo 'Unicorn will keep on working'",
                "CODE_DEPLOYMENT": "cd {{.Dir}}/current && git fetch --tags -q origin && git reset -q --hard {{.CommitSha}}\ncd {{.Dir}}/current && RAILS_ENV={{.RailsEnv}} bundle install --gemfile {{.Dir}}/current/Gemfile --path {{.Dir}}/shared/bundle --deployment --quiet --without development test",
                "POST_DEPLOYMENT": "sudo /etc/init.d/unicorn hot-reload"
              },
              "options": {
                "Dir": "/var/www/our-main-application",
                "RailsEnv": "production"
              }
            },
            {
              "name": "migrator",
              "script_templates": {
                "MIGRATE_DATABASE": "cd {{.Dir}} && RAILS_ENV={{.RailsEnv}} bundle exec rake db:migrate"
              },
              "options": {
                "Dir": "/var/www/our-main-application",
                "RailsEnv": "production"
              }
            }
          ]
        }
      ]
    }
  ]
}
```

### General Properties

* `ssl_enabled` - Turn this on if your Applikatoni instance is
  accessed via `https`.
* `host` - The host of your Applikatoni instance. Example:
  `applikatoni.shipping-company.com`
* `session_secret` - The secret for encrypt sessions in cookies. Use a
  generated, random secret.
* `oauth2_state_string` - A random, unguessable string to confirm that the
  Applikatoni instance is the one specified at GitHub.
* `github_client_id` - The client ID from your GitHub OAuth2 application.
* `github_client_secret` - The client secret from your GitHub OAuth2 application.
* `mandrill_api_key` - The API key of your [Mandrill](https://mandrillapp.com/)
  account. This is needed to send daily digest emails. If this is blank or left
out, no daily digest email will be sent.
* `applications` - An array of application configurations that Applikatoni can deploy.

### Application Properties

Inside the `applications` array applications need to be configured.

* `name` - The name of the application. Shows up in the web interface, notifications, and so on.
* `read_usernames` - An array of GitHub usernames. Users with these names have "read" access to the application on Applikatoni. They can only look at the deployment history, but cannot deploy.
* `github_owner` - The owner of the GitHub repository. It's the `company` in `github.com/company/rails-app`.
* `github_repo` - The name of the GitHub repository. It's the `rails-app` in `github.com/company/rails-app`.
* `github_branches` - An array of branch names. These branches will show up with their current status on the application page in Applikatoni to easily deploy them with a click.
* `travis_image_url` - The URL to the [Travis CI status image](http://docs.travis-ci.com/user/status-images/), including the token.
* `daily_digest_receivers` - An array of email adresses to which the daily digest should be sent (if `mandrill_api_key` is not set, no daily digest will be sent).
* `daily_digest_target` - The name of the `target` for which the daily digest should be sent. For example: if you have `test`, `staging` and `production` targets, it makes sense to only send out daily digest emails for `production`.

### Target Properties

Inside the `targets` array of each configured application the targets to which
Applikatoni can deploy need to be configured.

Examples for targets are `production`, `staging`, `testing`, and so on. Each
target has its own number of hosts.

* `name` - The name of the target.
* `deployment_user` - The user on the target hosts that has access via SSH.
* `deployment_ssh_key` - The private SSH key of the deployment user. The public key of the user _must_ be added to the hosts, so Applikatoni can access the host without password authenficiation
* `deploy_username` - An array of GitHub usernames. Users with these names have "deploy" access to this target.
* `bugsnag_api_key` - Your Bugsnag API key. If this is set, Applikatoni will notify Bugsnag about a deployment to this target after a successful deployment. **If this is left blank, Applikatoni will not notify NewRelic about deployments**.
* `flowdock_endpoint` - The Flowdock [Message URL](https://www.flowdock.com/api/messages) including the [auth](https://www.flowdock.com/api/authentication) information. Example: `https://deadbeefdeadbeef@api.flowdock.com/flows/acme/main/messages`. **If this is left blank, Applikatoni will not notify Flowdock about deployments**.
* `newrelic_api_key` - The NewRelic API key. If this and `newrelic_app_id` are set, Applikatoni will notify NewRelic about successful deployments. **If this is left blank, Applikatoni will not notify NewRelic about deployments**.
* `newrelic_app_id` - The NewRelic Application ID. If this and `newrelic_api_key` are set, Applikatoni will notify NewRelic about successful deployments. **If this is left blank, Applikatoni will not notify NewRelic about deployments**.
* `hosts` - An array of hosts, where each host needs the properties `name` and `roles`. Example:

            {
              "name": "webapp.staging.company.com:22",
              "roles": ["web", "migrator"]
            },
            {
              "name": "workers.staging.company.com:22",
              "roles": ["workers"]
            }

            IMPORTANT: The host name _must_ include the port!

* `default_stages` - An array of stage names. These get executed per default on each deployment, if nothing else is specified in the web interface. **Order is important! The order determines the deployment order!**
* `available_stages` - An array of all available stages. These are all the available stages that can be selected in the web interface. **Order is important! The order determines the deployment order!**
* `roles` - An array of roles. The names of these roles must match the role
  names specified for the `hosts`.

### Role Properties

* `name` - The name of this role. Examples: "worker-server", "webapp", "database".
* `options` - A hash of options. The keys are the names of available variables
  in the `script_templates`.
* `script_templates` - A hash where the keys are the name of the corresponding
  stage (and they _must_ match a name in `available_stages`, otherwise they
  won't get executed). The values are templates in the syntax of Go's
[text/template](http://golang.org/pkg/text/template/) package.

A small example illustrates how this works:

```json
{
  "name": "web",
  "script_templates": {
    "CHECK_CONNECTION": "test -d {{.Dir}}",
    "PRE_DEPLOYMENT": "echo 'Unicorn will keep on working'",
    "CODE_DEPLOYMENT": "cd {{.Dir}}/current && git fetch origin && git reset --hard {{.CommitSha}}\ncd {{.Dir}}/current && RAILS_ENV={{.RailsEnv}} bundle install --gemfile {{.Dir}}/current/Gemfile --path {{.Dir}}/shared/bundle --deployment --quiet --without development test",
    "POST_DEPLOYMENT": "sudo /etc/init.d/unicorn hot-reload"
  },
  "options": {
    "Dir": "/var/www/our-main-application",
    "RailsEnv": "production"
  }
}
```

When these stages are executed `{{.Dir}}` and `{{.RailsEnv}}` in the templates
are replaced with the values specified in `options`. This:

            cd {{.Dir}} && RAILS_ENV={{.RailsEnv}} bundle exec rake db:migrate

gets turned into this before execution:

            cd /var/www/our-main-application && RAILS_ENV=production bundle exec rake db:migrate

The other variable, `CommitSha`, is a **special variable**. It gets merged into
the `options` field when a deployment is started. You can see the other special
variables [here](https://github.com/applikatoni/applikatoni/blob/fc0fab6ca7445dc471406d9f6dd7e38e23a02cd5/models/deployment_config.go#L29-L34).

#### Script Templates

Script templates, after being fully rendered with the options passed in, are
executed **line by line**.

**Important:** Script templates do not keep state between stages and commands!
Even though Applikatoni re-uses the SSH connections to each host, for each
line a new SSH session is used.

That means, that the working directory needs to be set for each **line**.

This is **wrong**:

    "CODE_DEPLOYMENT": "cd {{.Dir}}/current\ngit fetch origin"

This is **correct**:

    "CODE_DEPLOYMENT": "cd {{.Dir}}/current && git fetch origin"

If one line in a template fails, the whole stage is considered failed.

# Testing

Make sure you have `sqlite3` and `goose` installed.

```
cd server
goose -env="test" up
cd ..
go test ./...
```

# Contributing

All contributions are welcome! Is the documentation lacking something? Did you
find a bug? Do you have a question about how Applikatoni works? Open a issue!

If you want to contribute code to Applikatoni, use the standard GitHub model of
contributing:

1. Fork the repository
2. Create a branch in which you'll work (e.g. `config_validation`)
3. Make your commits in this branch
4. Add yourself to `CHANGELOG.md`
5. Send a pull request to merge your branch into the master branch

Here is a great explanation of how to handle GitHub forks and Go import paths:
[Contributing to Open Source Git Repositories in Go](https://splice.com/blog/contributing-open-source-git-repositories-go/).

Make sure that you have the dependencies installed. See [Installation](#installation).

To get the tests running locally, you have to make sure that your test database
is migrated to the newest version. See [Testing](#testing).

Before sending a pull request, make sure that the tests are green and the build
runs fine:

1. `go test ./...`
2. `cd server && go build -o applikatoni .`

Make sure you run `go fmt` before committing changes!

Should you add a test? It depends. If it's easy to do: by all means, go ahead
and do it! But especially the code in the `server` package is not that testable,
so I'd understand if you won't add a test.

Then send a pull request. If any of this doesn't work and you don't know why:
send a pull request (or open a issue) and we'll see what we can do!

To get Applikatoni running locally see [Installation](#installation),
[Getting Started](#getting-started) and [Usage](#usage). You will also need a
`configuration.json` file to play around with -- see [Sample](#sample) for this.
(And use [GitHub developer applications](https://github.com/settings/developers)
to login locally).

If you want to contribute and don't know where to start, check out the issues
tagged with [help wanted](https://github.com/applikatoni/applikatoni/labels/help%20wanted).

If you're looking for easy issues, look no further than the issues tagged as
[easy](https://github.com/applikatoni/applikatoni/labels/easy).


# Is it production ready?

We've been using Applikatoni at [flinc](https://flinc.org) for over half a year
with multiple applications without any major problems. Several hundreds of
deployments by different users to a lot of servers.

# Authors

* [Thorsten Ball](https://github.com/mrnugget) ([@thorstenball](https://twitter.com/thorstenball))
* [Christian Bäuerlein](https://github.com/fabrik42) ([@fabrik42](https://twitter.com/fabrik42))

# License

MIT License. See [LICENSE](LICENSE).

For ease of use, Applikatoni server ships with
[goose](https://bitbucket.org/liamstask/goose). The license of goose is the MIT
license.

For an improved user experience, Applikatoni server ships with
[clickspark.js](https://github.com/ymc-thzi/clickspark.js). The license of
clickspark.js is the MIT license.

# Developed at & sponsored by

<p align="center">
  <a href="https://flinc.org">
    <img height="200" width="200" style="padding: 5px;" src="https://s3.eu-central-1.amazonaws.com/applikatoni/assets/flinc_logo.png">
  </a>
<p>
