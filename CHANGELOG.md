# Revel Changelog

Legend:

* [FTR] New feature
* [ENH] Enhancement to current functionality
* [BUG] Fixed bug
* [HOT] Hotfix


## Revel v0.9.2 (Feb 25, 2014)

* [HOT] - `revel new` bug (#522)


## Revel v0.9.1 (Feb 24, 2014)

* [HOT] - Booking sample error


## Revel v0.9.0 (Feb 24, 2014)

* [BUG] #504 - File access via URL security issue
* [BUG] #489 - Email validator bug
* [BUG] #475 - File watcher infinite loop
* [BUG] #333 - Extensions in routes break parameters
* [FTR] #472 - Support for 3rd part app skeletons
* [ENH] #512 - Per session expiration methods
* [ENH] #496 - Type check renderArgs[CurrentLocalRenderArg]
* [ENH] #490 - App.conf manual typo
* [ENH] #487 - Make files executable on `revel build`
* [ENH] #482 - Retain input values after form valdiation
* [ENH] #473 - OnAppStart documentation
* [ENH] #466 - JSON error template quoting fix
* [ENH] #464 - Remove unneeded trace statement
* [ENH] #457 - Remove unneeded trace


## Revel v0.8 (Jan 5, 2014)

* [BUG] #379 - HTTP 500 error for not found public path files
* [FTR] #424 - HTTP pprof support
* [FTR] #346 - Redis Cache support
* [FTR] #292 - SMTP Mailer
* [ENH] #443 - Validator constructors to improve `v.Check()` usage
* [ENH] #439 - Basic terminal output coloring
* [ENH] #428 - Improve error message for missing `RenderArg`
* [ENH] #422 - Route embedding for modules
* [ENH] #413 - App version variable
* [ENH] #153 - $GOPATH-wide file watching aka hot loading
