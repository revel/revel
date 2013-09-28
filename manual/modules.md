---
title: Modules
layout: manual
---

Modules are packages that can be plugged into an application. They allow sharing of controllers, views, assets, and other code between multiple Revel applications or from third-party sources.

The module should have the same layout as a Revel application. The "hosting" application will merge it in as follows:

1. Any templates in module/app/views will be added to the Template Loader search path
2. Any controllers in module/app/controllers will be treated as if they were in your application
3. The assets are made available, via a route action of the form `Static.ServeModule("modulename","public")`
4. Routes can be included in your application with a route line of `module:modulename`

### Enabling a module

In order to add a module to your app, add a line to `app.conf`:

	module.mymodulename = go/import/path/to/module

An empty import path disables the module:

	module.mymodulename =

For example, to enable the test runner module:

	module.testrunner = github.com/robfig/revel/modules/testrunner
