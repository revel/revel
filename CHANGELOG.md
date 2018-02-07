# CHANGELOG

## v0.19.0
# Release 0.19.0

# Maintenance Release

This release is focused on improving the security and resolving some issues. 

**There are no breaking changes from version 0.18**

[[revel/cmd](https://github.com/revel/cmd)]
* Improved vendor folder detection revel/cmd#117
* Added ordering of controllers so order remains consistent in main.go revel/cmd#112
* Generate same value of `AppVersion` regardless of where Revel is run revel/cmd#108
* Added referrer policy security header revel/cmd#114

[[revel/modules](https://github.com/revel/modules)]
* Added directory representation to static module revel/modules#46
* Gorp enhancements (added abstraction layer for transactions and database connection so both can be used), Added security fix for CSRF module revel/modules#68
* Added authorization configuration options to job page revel/modules#44

[[revel/examples](https://github.com/revel/examples)]
* General improvements and examples added revel/examples#39  revel/examples#40

## v0.18
# Release 0.18

## Upgrade path
The main breaking change is the removal of `http.Request` from the `revel.Request` object.
Everything else should just work....

## New items
* Server Engine revel/revel#998
The server engine implementation is described in the [docs](http://revel.github.io/manual/server-engine.html)
* Allow binding to a structured map. revel/revel#998 
Have a structure inside a map object which will be realized properly from params
* Gorm module revel/modules/#51
Added transaction controller
* Gorp module revel/modules/#52
* Autorun on startup in develop mode revel/cmd#95
Start the application without doing a request first using revel run ....
* Logger update revel/revel#1213
Configurable logger and added context logging on controller via controller.Log
* Before after finally panic controller method detection revel/revel#1211 
Controller methods will be automatically detected and called - similar to interceptors but without the extra code
* Float validation revel/revel#1209
Added validation for floats
* Timeago template function revel/revel#1207
Added timeago function to Revel template functions
* Authorization to jobs module revel/module#44
Added ability to specify authorization to access the jobs module routes
* Add MessageKey, ErrorKey methods to ValidationResult object revel/revel#1215
This allows the message translator to translate the keys added. So model objects can send out validation codes
* Vendor friendlier - Revel recognizes and uses `deps` (to checkout go libraries) if a vendor folder exists in the project root. 
* Updated examples to use Gorp modules and new loggers


### Breaking Changes

* `http.Request` is no longer contained in `revel.Request` revel.Request remains functionally the same but 
you cannot extract the `http.Request` from it. You can get the `http.Request` from `revel.Controller.Request.In.GetRaw().(*http.Request)`
* `http.Response.Out` Is not the http.Response and is deprecated, you can get the output writer by doing `http.Response.GetWriter()`. You can get the `http.Response` from revel.Controller.Response.Out.Server.GetRaw().(*http.Response)`

* `Websocket` changes. `revel.ServerWebsocket` is the new type of object you need to declare for controllers 
which should need to attach to websockets. Implementation of these objects have been simplified

Old
```

func (c WebSocket) RoomSocket(user string, ws *websocket.Conn) revel.Result {
	// Join the room.
	subscription := chatroom.Subscribe()
	defer subscription.Cancel()

	chatroom.Join(user)
	defer chatroom.Leave(user)

	// Send down the archive.
	for _, event := range subscription.Archive {
		if websocket.JSON.Send(ws, &event) != nil {
			// They disconnected
			return nil
		}
	}

	// In order to select between websocket messages and subscription events, we
	// need to stuff websocket events into a channel.
	newMessages := make(chan string)
	go func() {
		var msg string
		for {
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				close(newMessages)
				return
			}
			newMessages <- msg
		}
	}()
```
New
```
func (c WebSocket) RoomSocket(user string, ws revel.ServerWebSocket) revel.Result {
	// Join the room.
	subscription := chatroom.Subscribe()
	defer subscription.Cancel()

	chatroom.Join(user)
	defer chatroom.Leave(user)

	// Send down the archive.
	for _, event := range subscription.Archive {
		if ws.MessageSendJSON(&event) != nil {
			// They disconnected
			return nil
		}
	}

	// In order to select between websocket messages and subscription events, we
	// need to stuff websocket events into a channel.
	newMessages := make(chan string)
	go func() {
		var msg string
		for {
			err := ws.MessageReceiveJSON(&msg)
			if err != nil {
				close(newMessages)
				return
			}
			newMessages <- msg
		}
	}()
```
* GORM module has been refactored into modules/orm/gorm 


### Deprecated methods
* `revel.Request.FormValue()` Is deprecated, you should use methods in the controller.Params to access this data
* `revel.Request.PostFormValue()` Is deprecated, you should use methods in the controller.Params.Form to access this data
* `revel.Request.ParseForm()` Is deprecated - not needed
* `revel.Request.ParseMultipartForm()` Is deprecated - not needed
* `revel.Request.Form` Is deprecated, you should use the controller.Params.Form to access this data
* `revel.Request.MultipartForm` Is deprecated, you should use the controller.Params.Form to access this data
* `revel.TRACE`, `revel.INFO` `revel.WARN` `revel.ERROR` are deprecated. Use new application logger `revel.AppLog` and the controller logger `controller.Log`. See [logging](http://revel.github.io/manual/logging.html) for more details.

### Features

* Pluggable server engine support. You can now implement **your own server engine**. This means if you need to listen to more then 1 IP address or port you can implement a custom server engine to do this. By default Revel uses GO http server, but also available is fasthttp server in the revel/modules repository. See the docs for more information on how to implement your own engine.

### Enhancements
* Controller instances are cached for reuse. This speeds up the request response time and prevents unnecessary garbage collection cycles.  

### Bug fixes




## v0.17

[[revel/revel](https://github.com/revel/revel)]

* add-validation
* i18-lang-by-param
* Added namespace to routes, controllers
* Added go 1.6 to testing
* Adds the ability to set the language by a url parameter. The route file will need to specify the parameter so that it will be picked up
* Changed url validation logic to regex
* Added new validation mehtods (IPAddr,MacAddr,Domain,URL,PureText)

[[revel/cmd](https://github.com/revel/cmd)]

* no changes

[[revel/config](https://github.com/revel/config)]

* no changes

[[revel/modules](https://github.com/revel/modules)]

* Added Gorm module

[[revel/cron](https://github.com/revel/cron)]

* Updated cron task manager
* Added ability to run a specific job, reschedules job if cron is running.

[[revel/examples](https://github.com/revel/examples)]

* Gorm module (Example)

# v0.16.0

Deprecating support for golang versions prior to 1.6
### Breaking Changes

* `CurrentLocaleRenderArg` to `CurrentLocaleViewArg` for consistency
* JSON requests are now parsed by Revel, if the content type is `text/json` or `application/json`. The raw data is available in `Revel.Controller.Params.JSON`. But you can also use the automatic controller operation to load the data like you would any structure or map. See [here](http://revel.github.io/manual/parameters.html) for more details

### Features

* Modular Template Engine #1170 
* Pongo2 engine driver added revel/modules#39
* Ace engine driver added revel/modules#40
* Added i18n template support #746 

### Enhancements

* JSON request binding #1161 
* revel.SetSecretKey function added #1127 
* ResolveFormat now looks at the extension as well (this sets the content type) #936 
* Updated command to run tests using the configuration revel/cmd#61

### Bug fixes

* Updated documentation typos revel/modules#37
* Updated order of parameter map assignment #1155 
* Updated cookie lifetime for firefox #1174 
* Added test path for modules, so modules will run tests as well #1162 
* Fixed go profiler module revel/modules#20


# v0.15.0
@shawncatz released this on 2017-05-11

Deprecating support for golang versions prior to 1.7

### Breaking Changes

* None

### Features

* None

### Enhancements

* Update and improve docs revel/examples#17 revel/cmd#85

### Bug fixes

* Prevent XSS revel/revel#1153
* Improve error checking for go version detection revel/cmd#86

# v0.14.0
@notzippy released this on 2017-03-24

## Changes since v0.13.0

#### Breaking Changes
- `revel/revel`:
  - change RenderArgs to ViewArgs PR #1135
  - change RenderJson to RenderJSON PR #1057
  - change RenderHtml to RenderHTML PR #1057
  - change RenderXml to RenderXML PR #1057

#### Features
- `revel/revel`:

#### Enhancements
- `revel/revel`:


#### Bug Fixes
- `revel/revel`:


# v0.13.1
@jeevatkm released this on 2016-06-07

**Bug fix:**
- Windows path fix #1064


# v0.13.0
@jeevatkm released this on 2016-06-06

## Changes since v0.12.0

#### Breaking Changes
- `revel/revel`:
  - Application Config name changed from `watcher.*` to `watch.*`  PR #992, PR #991

#### Features
- `revel/revel`:
  - Request access log PR #1059, PR #913, #1055
  - Messages loaded from modules too PR #828
- `revel/cmd`:
  - Added `revel version` command emits the revel version and go version revel/cmd#19

#### Enhancements
- `revel/revel`:
  - Creates log directory if missing PR #1039
  - Added `application/javascript` to accepted headers PR #1022
  - You can change `Server.Addr` value via hook function PR #999
  - Improved deflate/gzip compressor PR #995
  - Consistent config name `watch.*` PR #992, PR #991
  - Defaults to HttpOnly and always secure cookies for non-dev mode #942, PR #943
  - Configurable server Read and Write Timeout via app config #936, PR #940
  - `OnAppStart` hook now supports order param too PR #935
  - Added `PutForm` and `PutFormCustom` helper method in `testing.TestSuite` #898
  - Validator supports UTF-8 string too PR #891, #841
  - Added `InitServer` method that returns `http.HandlerFunc` PR #879
  - Symlink aware processing Views, Messages and Watch mode PR #867, #673
  - Added i18n settings support unknown format PR #852
  - i18n: Make Message Translation pluggable PR #768
  - jQuery `min-2.2.4` & Bootstrap `min-3.3.6` version updated in `skeleton/public` #1063
- `revel/cmd`:
  - Revel identifies current `GOPATH` and performs `new` command; relative to directory revel/revel#1004
  - Installs package dependencies during a build PR revel/cmd#43
  - Non-200 response of test case request will correctly result into error PR revel/cmd#38
  - Websockets SSL support in `dev` mode PR revel/cmd#32
  - Won't yell about non-existent directory while cleaning PR revel/cmd#31, #908
    - [x] non-fatal errors when building #908
  - Improved warnings about route generation PR revel/cmd#25
  - Command is Symlink aware PR revel/cmd#20
  - `revel package` & `revel build` now supports environment mode PR revel/cmd#14
  - `revel clean` now cleans generated routes too PR revel/cmd#6
- `revel/config`:
  - Upstream `robfig/config` refresh and import path updated from `github.com/revel/revel/config` to `github.com/revel/config`, PR #868
  - Config loading order and external configuration to override application configuration revel/config#4 [commit](https://github.com/revel/revel/commit/f3a422c228994978ae0a5dd837afa97248b26b41)
  - Application config error will produce insight on error PR revel/config#3 [commit](https://github.com/revel/config/commit/85a123061070899a82f59c5ef6187e8fb4457f64)
- `revel/modules`:
  - Testrunner enhancements
    - Minor improvement on testrunner module PR #820, #895
    - Add Test Runner panels per test group PR revel/modules#12
- `revel/revel.github.io`:
  - Update `index.md` and homepage (change how samples repo is installed) PR [#85](https://github.com/revel/revel.github.io/pull/85)
  - Couple of UI improvements PR [#93](https://github.com/revel/revel.github.io/pull/93)
  - Updated techempower benchmarks Round 11 [URL](http://www.techempower.com/benchmarks/#section=data-r11)
  - Docs updated for v0.13 release
- Cross-Platform Support
  - Slashes should be normalized in paths #260, PR #1028, PR #928

#### Bug Fixes
- `revel/revel`:
  - Binder: Multipart `io.Reader` parameters needs to be closed #756
  - Default Date & Time Format correct in skeleton PR #1062, #878
  - Addressed with alternative for `json: unsupported type: <-chan struct {}` on Go 1.6 revel/revel#1037
  - Addressed one edge case, invalid Accept-Encoding header causes panic revel/revel#914


# v0.11.3
@brendensoares released this on 2015-01-04

This is a minor release to address a critical bug (#824) in v0.11.2.

Everybody is strongly encouraged to rebuild their projects with the latest version of Revel. To do it, execute the commands:

``` sh
$ go get -u github.com/revel/cmd/revel

$ revel build github.com/myusername/myproject /path/to/destination/folder
```


# v0.11.2
on 2014-11-23

This is a minor release to address a critical bug in v0.11.0.

Everybody is strongly encouraged to rebuild their projects with the latest version of Revel. To do it, execute the commands:

``` sh
$ go get -u github.com/revel/cmd/revel

$ revel build github.com/myusername/myproject /path/to/destination/folder
```


# v0.11.1
@pushrax released this on 2014-10-27

This is a minor release to address a compilation error in v0.11.0.


# v0.12.0
@brendensoares released this on 2015-03-25

Changes since v0.11.3:

## Breaking Changes
1. Add import path to new `testing` sub-package for all Revel tests. For example:

``` go
package tests

import "github.com/revel/revel/testing"

type AppTest struct {
    testing.TestSuite
}
```
1. We've relocated modules to a dedicated repo. Make sure you update your `conf/app.conf`. For example, change:

``` ini
module.static=github.com/revel/revel/modules/static
module.testrunner = github.com/revel/revel/modules/testrunner
```

to the new paths:

``` ini
module.static=github.com/revel/modules/static
module.testrunner = github.com/revel/modules/testrunner
```

## [ROADMAP] Focus: Improve Internal Organization

The majority of our effort here is increasing the modularity of the code within Revel so that further development can be done more productively while keeping documentation up to date.
- `revel/revel.github.io`
  - [x] Improve docs #[43](https://github.com/revel/revel.github.io/pull/43)
- `revel/revel`:
  - [x] Move the `revel/revel/harness` to the `revel/cmd` repo since it's only used during build time. #[714](https://github.com/revel/revel/issues/714)
  - [x] Move `revel/revel/modules` to the `revel/modules` repo #[785](https://github.com/revel/revel/issues/785)
  - [x] Move `revel/revel/samples` to the `revel/samples` repo #[784](https://github.com/revel/revel/issues/784)
  - [x] `testing` TestSuite #[737](https://github.com/revel/revel/issues/737) #[810](https://github.com/revel/revel/issues/810)
  - [x] Feature/sane http timeout defaults #[837](https://github.com/revel/revel/issues/837) PR#[843](https://github.com/revel/revel/issues/843) Bug Fix PR#[860](https://github.com/revel/revel/issues/860)
  - [x] Eagerly load templates in dev mode #[353](https://github.com/revel/revel/issues/353) PR#[844](https://github.com/revel/revel/pull/844)
  - [x] Add an option to trim whitespace from rendered HTML #[800](https://github.com/revel/revel/issues/800)
  - [x] Remove built-in mailer in favor of 3rd party package #[783](https://github.com/revel/revel/issues/783)
  - [x] Allow local reverse proxy access to jobs module status page for IPv4/6 #[481](https://github.com/revel/revel/issues/481) PR#[6](https://github.com/revel/modules/pull/6) PR#[7](https://github.com/revel/modules/pull/7)
  - [x] Add default http.Status code for render methods. #[728](https://github.com/revel/revel/issues/728)
  - [x] add domain for cookie #[770](https://github.com/revel/revel/issues/770) PR#[882](https://github.com/revel/revel/pull/882)
  - [x] production mode panic bug #[831](https://github.com/revel/revel/issues/831) PR#[881](https://github.com/revel/revel/pull/881)
  - [x] Fixes template loading order whether watcher is enabled or not #[844](https://github.com/revel/revel/issues/844)
  - [x] Fixes reverse routing wildcard bug PR#[886](https://github.com/revel/revel/pull/886) #[869](https://github.com/revel/revel/issues/869)
  - [x] Fixes router app start bug without routes. PR #[855](https://github.com/revel/revel/pull/855)
  - [x] Friendly URL template errors; Fixes template `url` func "index out of range" when param is `undefined` #[811](https://github.com/revel/revel/issues/811) PR#[880](https://github.com/revel/revel/pull/880)
  - [x] Make result compression conditional PR#[888](https://github.com/revel/revel/pull/888)
  - [x] ensure routes are loaded before returning from OnAppStart callback PR#[884](https://github.com/revel/revel/pull/884)
  - [x] Use "302 Found" HTTP code for redirect PR#[900](https://github.com/revel/revel/pull/900)
  - [x] Fix broken fake app tests PR#[899](https://github.com/revel/revel/pull/899)
  - [x] Optimize search of template names PR#[885](https://github.com/revel/revel/pull/885)
- `revel/cmd`:
  - [x] track current Revel version #[418](https://github.com/revel/revel/issues/418) PR#[858](https://github.com/revel/revel/pull/858)
  - [x] log path error After revel build? #[763](https://github.com/revel/revel/issues/763)
  - [x] Use a separate directory for revel project binaries #[17](https://github.com/revel/cmd/pull/17) #[819](https://github.com/revel/revel/issues/819)
  - [x] Overwrite generated app files instead of deleting directory #[551](https://github.com/revel/revel/issues/551) PR#[23](https://github.com/revel/cmd/pull/23)
- `revel/modules`:
  - [x] Adds runtime pprof/trace support #[9](https://github.com/revel/modules/pull/9)
- Community Goals:
  - [x] Issue labels #[545](https://github.com/revel/revel/issues/545)
    - [x] Sync up labels/milestones in other repos #[721](https://github.com/revel/revel/issues/721)
  - [x] Update the Revel Manual to reflect current features
    - [x] [revel/revel.github.io/32](https://github.com/revel/revel.github.io/issues/32)
    - [x] [revel/revel.github.io/39](https://github.com/revel/revel.github.io/issues/39)
    - [x] Docs are obsolete, inaccessible TestRequest.testSuite #[791](https://github.com/revel/revel/issues/791)
    - [x] Some questions about revel & go docs #[793](https://github.com/revel/revel/issues/793)
  - [x] RFCs to organize features #[827](https://github.com/revel/revel/issues/827)

[Full list of commits](https://github.com/revel/revel/compare/v0.11.3...v0.12.0)


# v0.11.0
@brendensoares released this on 2014-10-26

Note, Revel 0.11 requires Go 1.3 or higher.

Changes since v0.10:

[BUG]   #729    Adding define inside the template results in an error (Changes how template file name case insensitivity is handled)

[ENH]   #769    Add swap files to gitignore
[ENH]   #766    Added passing in build flags to the go build command
[ENH]   #761    Fixing cross-compiling issue #456 setting windows path from linux
[ENH]   #759    Include upload sample's tests in travis
[ENH]   #755    Changes c.Action to be the action method name's letter casing per #635
[ENH]   #754    Adds call stack display to runtime panic in browser to match console
[ENH]   #740    Redis Cache: Add timeouts.
[ENH]   #734    watcher: treat fsnotify Op as a bitmask
[ENH]   #731    Second struct in type revel fails to find the controller
[ENH]   #725    Testrunner: show response info
[ENH]   #723    Improved compilation errors and open file from error page
[ENH]   #720    Get testrunner path from config file
[ENH]   #707    Add log.colorize option to enable/disable colorize
[ENH]   #696    Revel file upload testing
[ENH]   #694    Install dependencies at build time
[ENH]   #693    Prefer extension over Accept header
[ENH]   #692    Update fsnotify to v1 API
[ENH]   #690    Support zero downtime restarts
[ENH]   #687    Tests: request override
[ENH]   #685    Persona sample tests and bugfix
[ENH]   #598    Added README file to Revel skeleton
[ENH]   #591    Realtime rebuild
[ENH]   #573    Add AppRoot to allow changing the root path of an application

[FTR]   #606    CSRF Support

[Full list of commits](https://github.com/revel/revel/compare/v0.10.0...v0.11.0)


# v0.10.0
@brendensoares released this on 2014-08-10

Changes since v0.9.1:
- [FTR] #641 - Add "X-HTTP-Method-Override" to router
- [FTR] #583 - Added HttpMethodOverride filter to routes
- [FTR] #540 - watcher flag for refresh on app start
- [BUG] #681 - Case insensitive comparison for websocket upgrades (Fixes IE Websockets ...
- [BUG] #668 - Compression: Properly close gzip/deflate
- [BUG] #667 - Fix redis GetMulti and improve test coverage
- [BUG] #664 - Is compression working correct?
- [BUG] #657 - Redis Cache: panic when testing Ge
- [BUG] #637 - RedisCache: fix Get/GetMulti error return
- [BUG] #621 - Bugfix/router csv error
- [BUG] #618 - Router throws exception when parsing line with multiple default string arguments
- [BUG] #604 - Compression: Properly close gzip/deflate.
- [BUG] #567 - Fixed regex pattern to properly require message files to have a dot in filename
- [BUG] #566 - Compression fails ("unexpected EOF" in tests)
- [BUG] #287 - Don't remove the parent folders containing generated code.
- [BUG] #556 - fix for #534, also added url path to not found message
- [BUG] #534 - Websocket route not found
- [BUG] #343 - validation.Required(funtionCall).Key(...) - reflect.go:715: Failed to generate name for field.
- [ENH] #643 - Documentation Fix in Skeleton for OnAppStart
- [ENH] #674 - Removes custom `eq` template function
- [ENH] #669 - Develop compress closenotifier
- [ENH] #663 - fix for static content type not being set and defaulting to OS
- [ENH] #658 - Minor: fix niggle with import statement
- [ENH] #652 - Update the contributing guidelines
- [ENH] #651 - Use upstream gomemcache again
- [ENH] #650 - Go back to upstream memcached library
- [ENH] #612 - Fix CI package error
- [ENH] #611 - Fix "go vet" problems
- [ENH] #610 - Added MakeMultipartRequest() to the TestSuite
- [ENH] #608 - Develop compress closenotifier
- [ENH] #596 - Expose redis cache options to config
- [ENH] #581 - Make the option template tag type agnostic.
- [ENH] #576 - Defer session instantiation to first set
- [ENH] #565 - Fix #563 -- Some custom template funcs cannot be used in JavaScript cont...
- [ENH] #563 - TemplateFuncs cannot be used in JavaScript context
- [ENH] #561 - Fix missing extension from message file causing panic
- [ENH] #560 - enhancement / templateFunc `firstof`
- [ENH] #555 - adding symlink handling to the template loader and watcher processes
- [ENH] #531 - Update app.conf.template
- [ENH] #520 - Respect controller's Response.Status when action returns nil
- [ENH] #519 - Link to issues
- [ENH] #486 - Support for json compress
- [ENH] #480 - Eq implementation in template.go still necessary ?
- [ENH] #461 - Cron jobs not started until I pull a page
- [ENH] #323 - disable session/set-cookie for `Static.Serve()`

[Full list of commits](https://github.com/revel/revel/compare/v0.9.1...v0.10.0)


# v0.9.1
@pushrax released this on 2014-03-02

Minor patch release to address a couple bugs.

Changes since v0.9.0:
- [BUG] #529 - Wrong path was used to determine existence of `.git`
- [BUG] #532 - Fix typo for new type `ValidEmail`

The full list of commits can be found [here](https://github.com/revel/revel/compare/v0.9.0...v0.9.1).


# v0.9.0
@pushrax released this on 2014-02-26

## Revel GitHub Organization

We've moved development of the framework to the @revel GitHub organization, to help manage the project as Revel grows. The old import path is still valid, but will not be updated in the future.

You'll need to manually update your apps to work with the new import path. This can be done by replacing all instances of `github.com/robfig/revel` with `github.com/revel/revel` in your app, and running:

```
$ cd your_app_folder
$ go get -u github.com/howeyc/fsnotify  # needs updating
$ go get github.com/revel/revel
$ go get github.com/revel/cmd/revel     # command line tools have moved
```

**Note:** if you have references to `github.com/robfig/revel/revel` in any files, you need to replace them with `github.com/revel/cmd/revel` _before_ replacing `github.com/robfig/revel`! (note the prefix collision)

If you have any trouble upgrading or notice something we missed, feel free to hop in the IRC channel (#revel on Freenode) or send the mailing list a message.

Also note, the documentation is now at [revel.github.io](http://revel.github.io)!

Changes since v0.8:
- [BUG] #522 - `revel new` bug
- [BUG] - Booking sample error
- [BUG] #504 - File access via URL security issue
- [BUG] #489 - Email validator bug
- [BUG] #475 - File watcher infinite loop
- [BUG] #333 - Extensions in routes break parameters
- [FTR] #472 - Support for 3rd part app skeletons
- [ENH] #512 - Per session expiration methods
- [ENH] #496 - Type check renderArgs[CurrentLocalRenderArg]
- [ENH] #490 - App.conf manual typo
- [ENH] #487 - Make files executable on `revel build`
- [ENH] #482 - Retain input values after form valdiation
- [ENH] #473 - OnAppStart documentation
- [ENH] #466 - JSON error template quoting fix
- [ENH] #464 - Remove unneeded trace statement
- [ENH] #457 - Remove unneeded trace
- [ENH] #508 - Support arbitrary network types
- [ENH] #516 - Add Date and Message-Id mail headers

The full list of commits can be found [here](https://github.com/revel/revel/compare/v0.8...v0.9.0).


# v0.8
@pushrax released this on 2014-01-06

Changes since v0.7:
- [BUG] #379 - HTTP 500 error for not found public path files
- [FTR] #424 - HTTP pprof support
- [FTR] #346 - Redis Cache support
- [FTR] #292 - SMTP Mailer
- [ENH] #443 - Validator constructors to improve `v.Check()` usage
- [ENH] #439 - Basic terminal output coloring
- [ENH] #428 - Improve error message for missing `RenderArg`
- [ENH] #422 - Route embedding for modules
- [ENH] #413 - App version variable
- [ENH] #153 - $GOPATH-wide file watching aka hot loading


# v0.6
@robfig released this on 2013-09-16



