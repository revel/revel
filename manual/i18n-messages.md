---
title: Messages
layout: manual
---

Messages are used to externalize pieces of text in order to be able to provide translations for them. Revel
supports message files organized per locale (a combination of *language* and *region*), transparent locale 
look-up, cookie-based overrides and message nesting and arguments.

#### Glossary
* Locale: a combination of *language* and *region* that indicates a user language preference, eg. `en-US`.
* Language: the language part of a locale, eg. `en`. Language identifiers are expected to be [ISO 639-1 codes](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Region: the region part of a locale, eg. `US`. Region identifiers are expected to be [ISO 3166-1 alpha-2 codes](http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2).

# Quick start
* Message files are UTF-8 encoded files stored in the `messages` folder in the root your application.
* The *file extension* determines the *language* of the text contained in the file and should be an [ISO 639-1 code](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Each message file contains the key-value pairs that can be used throughout your application.
* Each message file can contain any number of *region* sections (identified by a [ISO 3166-1 alpha-2 code](http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)) that allow you to override messages in the file on a per-region basis.
* Each message file is effectively a [goconfig file](https://github.com/robfig/goconfig) and supports all goconfig features.

#### Sample application
The way Revel handles message files and internationalization in general is similar to most other web frameworks out there. For those of you that wish to get
started straight away without going through the specifics, there is a sample application `revel/samples/i18n` that you can have a look at which demonstrates 
all the basics.

# Reference

The following chapters will describe each part of the framework in detail.

## Message files

Message files is the central concept of internationalized messages in Revel. They contain the actual text that will be used while rendering the view (or 
elsewhere in your application if you so desire). When creating new message files, there are a couple of rules to keep in mind:

