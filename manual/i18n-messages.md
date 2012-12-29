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

## Quick start

### Sample application
The way Revel handles message files and internationalization in general is similar to most other web frameworks out there. For those of you that wish to get
started straight away without going through the specifics, there is a sample application `revel/samples/i18n` that you can have a look at which demonstrates 
all the basics.

### Summary
* Message files are UTF-8 encoded files stored in the `messages` folder in the root your application.
* The *file extension* determines the *language* of the text contained in the file and should be an [ISO 639-1 code](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Each message file contains the key-value pairs that can be used throughout your application.
* Each message file can contain any number of *region* sections (identified by a [ISO 3166-1 alpha-2 code](http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)) that allow you 
to override messages in the file on a per-region basis.
* Each message file is effectively a [goconfig file](https://github.com/robfig/goconfig) and supports all goconfig features.

The following chapters will describe each part of the framework in detail.

## Message files

Message files is the central concept of internationalized messages in Revel. They contain the actual text that will be used while rendering the view (or 
elsewhere in your application if you so desire). When creating new message files, there are a couple of rules to keep in mind:

* All message files should be stored in the `messages` folder in the application root.
* The file extension determines the *language* of the message file and should be an [ISO 639-1 code](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Message files should be UTF-8 encoded. While this is not a hard requirement, it is best practice.

### Organizing message files

There are no restrictions on message file names; a message file name can be anything as long as it has a valid extention. There is also no restriction on the *amount*
of files per language. When the application starts, Revel will walk all message files with a valid extension in the `messages` folder and merge them according to their 
language. This means that you are free to organize the message files however you want.

For example, you may want to take a traditional approach and define 1 single message file per language:

    /messages
        messages.en
        messages.fr
        ...

Another approach would be to create *multiple files* for the *same language* and organize them based on the kind of messages they contain:

    /messages
        labels.en
        warnings.en
        labels.fr
        warnings.fr
        ...

**Important note:** while it's technically possible to define the same *message key* in multiple files with the same language, this will result in unpredictable behaviour.
When using multiple files per language, take care to keep your message keys unique so that keys in one file cannot be overwritten after merging!

### Message keys and values

A message file is for all intents and purposes a [goconfig file](https://github.com/robfig/goconfig). This means that messages should be defined according to the tried and
tested key-value format:

    key=value

For example:

    greeting=Hello 
    greeting.name=Rob
    greeting.suffix=, welcome to Revel!

### Sections

A goconfig file is separated into *sections*. The *default section* always exists and contains any messages that are not defined in a specific section. For example:

    key=value

    [SECTION]
    key2=value2

The `key=value` message is implicitly put in the default section as it was not defined under another specific section.

For message files all messages should be defined in the *default section* unless they are specific to a certain region (see 
[Sections and regions](#regions) for more information).

### Regions

Region-specific messages should be defined in sections with the same name. For example, suppose that we want to greet all English speaking users with `"Hello"`, all British
users with `"Hey"` and all American users with `"Howdy"`. In order to accomplish this, we could define the following message file `greeting.en`:

    greeting=Hello

    [GB]
    greeting=Hey

    [US]
    greeting=Howdy

For users who have defined English (`en`) as their preferred language, Revel would resolve `greeting` to `Hello`. Only in specific cases where the user's locale has been
explicitly defined as `en-GB` or `en-US` would the `greeting` message be resolved using the specific sections.

**Important note:** messages defined under a section that is not a valid region are technically allowed but ultimately useless (as they will never be resolved).

### Referencing and arguments

#### Referencing

Messages in message files can reference eachother. This allows users to compose a single message from multiple other messages. The syntax for referencing other messages is
`%(key)s`. For example:

    greeting=Hello 
    greeting.name=Rob
    greeting.suffix=, welcome to Revel!
    greeting.full=%(greeting)s %(greeting.name)s%(greeting.suffix)s

*Note:* referencing is a [goconfig file](https://github.com/robfig/goconfig) feature.

#### Arguments

Messages support one or more arguments. Arguments in messages are resolved using the same rules as the go `fmt` package. For example:

    greeting.name_arg=Hello %s!

## Resolving messages

Messages can be resolved from either a *view template* or a *controller*.

### Controller

Each controller has a convenience function `Message(...)` that can be used to resolve messages.

...

### Template

...

## Resolving the client locale

...

* Cookie
* Accept-Language HTTP header
* Default language

...