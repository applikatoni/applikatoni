# CHANGELOG

## Unreleased

* Fix the indentation of multi-line deployment comments in Flowdock summary (mrnugget)
* **BREAKING CHANGE**: fix unstable webhooks and "database is locked" errors by
  reducing the amount of webhook triggers. Webhooks now only receive a maximum
  of 3 requests per deployment, corresponding to the state changes of a
  deployment. See [these constants](https://github.com/applikatoni/applikatoni/blob/15d2e6b9f7c4f581ca6619e8e3015a53b541ed9a/models/deployment.go#L8-L11)
  for possible states. See [these structs](https://github.com/applikatoni/applikatoni/blob/9eeb547d140f8dcc0358b1e6f98fd36bdfe23fd6/server/webhook_notifier.go#L14-L50)
  for the new schema of a webhook payload. Webhooks are now _only_ URLs and which
  webhook receives a request on which payload cannot be configured anymore. See
  [Issue #35](https://github.com/applikatoni/applikatoni/issues/35) for more
  information on the topic. (PR 36, mrnugget)

## 1.3.0 - 18. January 2016

* Fix nil-pointer dereference in notifiers when `err` is set and
  `resp.StatusCode` is being accessed (Issue 33, mrnugget)
* Add generic webhooks that Applikatoni sends a payload to via HTTP to
  on certain events (PR 31, wind0r)
* Add ASCII banner (Issue 28, wind0r)

Diff: https://github.com/applikatoni/applikatoni/compare/1.2.1...1.3.0

## 1.2.1 - 18. December 2015

* Fix race-condition when generating summaries for Slack/Flowdock/NewRelic (mrnugget)

Diff: https://github.com/applikatoni/applikatoni/compare/1.2.0...1.2.1

## 1.2.0 - 15. December 2015

* Add Slack Integration (Issue 18, HParker)
* Add footer (Issue 17, HParker)
* Add ability to filter the list of deployments by target (Issue 9, wind0r)
* Stop breaking the layout of deployment overview with long deployment comments (Issue 10, wind0r)
* Add SHA and link to "what will be deployed"-diff (Issue 11, wind0r)
* Use markdown formatting for flowdock notifications (fabrik42)
* Ensure the database is migrated to newest version when booting up (Issue 2, wind0r)

Diff: https://github.com/applikatoni/applikatoni/compare/1.1.0...1.2.0

## 1.1.0 - 27. October 2015

* Show a diff between the current commit on the specified target and the
  selected commit before deploying code (mrnugget)
* Fix the broken redirect to `/login` when unauthorized (mrnugget)
* Do not allow empty deployment comments (mrnugget)
* Send correct `repository` value to Bugsnag (mrnugget)
* Fix timestamp in title attribute on deployment details page (mrnugget)
* Fix linebreaks in Flowdock messages (mrnugget)
* Fix deprecated import path of `uuid` package (Issue 1, mrnugget)

Diff: https://github.com/applikatoni/applikatoni/compare/1.0.0...1.1.0

## 1.0.0 - 09. September 2015

* Initial release
