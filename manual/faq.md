---
title: Frequently Asked Questions
layout: manual
---

> What is the relationship between interceptors, plugins, and modules? 

1. Modules are packages that can be plugged into an application. They allow sharing of controllers, views, assets, and other code between multiple Revel applications (or from third-party sources). 
2. Plugins are types that may be registered to hook into application and request lifecycle events.  They apply to the application as a whole, unlike Interceptors which generally apply to a specific Controller.
3. Interceptors are functions that may be registered to hook into specific controllers' request lifecycle events.
