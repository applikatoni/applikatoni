![Applikatoni - Deployments Al Forno](https://s3.eu-central-1.amazonaws.com/applikatoni/assets/banner_with_slogan.png)
[![Build Status](https://travis-ci.org/applikatoni/applikatoni.svg?branch=master)](https://travis-ci.org/applikatoni/applikatoni)

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

2. Download the Applikatoni server package and extract it. Or install from
   source. See the [README](https://github.com/applikatoni/applikatoni/tree/master/server) for more information.

    Version 0.1: [OS X](https://s3.eu-central-1.amazonaws.com/applikatoni/builds/applikatoni-darwin-amd64-1440229141.tar.gz)

3. Configure Applikatoni. Check out the [README](./server/README.md) of the
   Applikatoni server for detailed instructions.

4. Setup the database with the packaged [goose](https://bitbucket.org/liamstask/goose) binary.

5. Start Applikatoni

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
