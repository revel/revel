---
title: Frequently Asked Questions
layout: manual
---

> What is the relationship between interceptors, filters, and modules?

1. Modules are packages that can be plugged into an application. They allow
sharing of controllers, views, assets, and other code between multiple Revel
applications (or from third-party sources).

2. Filters are functions that may be hooked into the request processing
pipeline.  They generally apply to the application as a whole and handle
technical concerns, orthogonal to application logic.

3. Interceptors are a convenient way to package data and behavior, since
embedding a type imports its interceptors and fields.  This makes interceptors
useful for things like verifying the login cookie and saving that information
into a field.
