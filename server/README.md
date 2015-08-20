# applikatoni - The Applikatoni server

## Dependencies

* sqlite3
* goose - https://bitbucket.org/liamstask/goose/

## Installation
### Download a packaged version

* TODO: Links to packaged versions

### Building from source

1. Set up the repository inside your Go workspace:

        mkdir -p $GOPATH/src/github.com/flinc/
        cd $GOPATH/src/github.com/flinc/
        git clone git@github.com:flinc/applikatoni.git
        cd applikatoni
2. Install dependencies:

        go get ./...
3. Build and run it

        go build -o server && ./server -port=":3000"

## Usage

1. Make sure the database file is setup and migrated:

        go get bitbucket.org/liamstask/goose/cmd/goose
        vim db/dbconf.yml
        goose -env="production" up
2. Create a `configuration.json` file for your needs. See [Configuration](##
   Configuration) for more information.

        cp configuration_example.json configuration.json
        vim configuration.json
3. Start the server:

        ./applikatoni -port=":8080" -db=./db/production.db -conf=./configuration.json

## Configuration

### Requirements
* a user on the servers you want to deploy your application to with a SSH-Key
* a GitHub OAuth application ([See here](https://github.com/settings/applications/new))

### The configuration.json

The Applikatoni is configured by reading a `configuration.json` file.

You can find a simplified example for a Rails application with two Unicorn web
servers and one Sidekiq worker server here:
[configuration_example.json](./configuration_example.json). Or read on to get a
run down of what it's doing.

#### Sample

Here is a sample `configuration.json` for an application called
`our-main-application` hosted under
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

#### General Properties

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

#### Application Properties

Inside the `applications` array applications need to be configured.

* `name` - The name of the application. Shows up in the web interface, notifications, and so on.
* `read_usernames` - An array of GitHub usernames. Users with these names have "read" access to the application on Applikatoni. They can only look at the deployment history, but cannot deploy.
* `github_owner` - The owner of the GitHub repository. It's the `company` in `github.com/company/rails-app`.
* `github_repo` - The name of the GitHub repository. It's the `rails-app` in `github.com/company/rails-app`.
* `github_branches` - An array of branch names. These branches will show up with their current status on the application page in Applikatoni to easily deploy them with a click.
* `travis_image_url` - The URL to the [Travis CI status image](http://docs.travis-ci.com/user/status-images/), including the token.
* `daily_digest_receivers` - An array of email adresses to which the daily digest should be sent (if `mandrill_api_key` is not set, no daily digest will be sent).
* `daily_digest_target` - The name of the `target` for which the daily digest should be sent. For example: if you have `test`, `staging` and `production` targets, it makes sense to only send out daily digest emails for `production`.

#### Target Properties

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

#### Role Properties

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
variables [here](https://github.com/flinc/applikatoni/blob/84946fcf6230fbcd03c2da52bf3d9002a12573d1/models/deployment_config.go#L29-L34).

##### Script Templates

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

## Testing

Make sure you have `sqlite3` and `goose` installed.

```
cd server
goose -env="test" up
cd ..
go test ./...
```
