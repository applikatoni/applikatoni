# CHANGELOG

## Unreleased

* Ensure the database is migrated to newest version when booting up (Issue 2, wind0r)

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
