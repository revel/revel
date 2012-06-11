---
title: Concepts
layout: manual
---

## MVC

Revel makes it easy to build web applications using the Model-View-Controller
(MVC) pattern by relying on conventions that require a certain structure in your
application.  In return, it is very light on configuration and enables an
extremely fast development cycle.

Here is a quick summary:

- *Models* are the essential data objects that describe your application domain.
   Models also contain domain-specific logic for querying and updating the data.
- *Views* describe how data is presented and manipulated. In our case, this is
   the template that is used to present data and controls to the user.
- *Controllers* handle the request execution.  They perform the user's desired
   action, they decide which View to display, and they prepare and provide the
   necessary data to the View for rendering.

There are many excellent overviews of MVC structure online.  In particular, the
one provided by [Play! Framework](http://www.playframework.org) matches our model exactly.

## Controllers and Actions

Each HTTP request invokes an *action*, which handles the request and writes the
response.

*Actions* are grouped into *Controllers*.


