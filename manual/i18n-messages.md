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

***

## Sample application

The way Revel handles message files and internationalization in general is similar to most other web frameworks out there. For those of you that wish to get
started straight away without going through the specifics, there is a sample application `revel/samples/i18n` that you can have a look at which demonstrates 
all the basics.

***

## Message files

Messages are defined in message files. These files contain the actual text that will be used while rendering the view (or elsewhere in your application if you so desire). 
When creating new message files, there are a couple of rules to keep in mind:

* All message files should be stored in the `messages` folder in the application root.
* The file extension determines the *language* of the message file and should be an [ISO 639-1 code](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Message files should be UTF-8 encoded. While this is not a hard requirement, it is best practice.
* Each message file is effectively a [goconfig file](https://github.com/robfig/goconfig) and supports all goconfig features.

### Organizing message files

There are no restrictions on message file names; a message file name can be anything as long as it has a valid extention. There is also no restriction on the *amount*
of files per language. When the application starts, Revel will parse all message files with a valid extension in the `messages` folder and merge them according to their 
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

<div class="alert alert-block"><strong>Important note:</strong> while it's technically possible to define the same <em>message key</em> in multiple files with the same language, this will result in unpredictable behaviour. When using multiple files per language, take care to keep your message keys unique so that keys in one file cannot be overwritten after the files are merged!</div>

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

<div class="alert alert-info"><strong>Note:</strong> sections are a <em>goconfig</em> feature.</div>

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

<div class="alert alert-block"><strong>Important note:</strong> messages defined under a section that is not a valid region are technically allowed but ultimately useless (as they will never be resolved).</div>

### Referencing and arguments

#### Referencing

Messages in message files can reference other messages. This allows users to compose a single message from multiple other messages. The syntax for referencing other messages 
is `%(key)s`. For example:

    greeting=Hello 
    greeting.name=Rob
    greeting.suffix=, welcome to Revel!
    greeting.full=%(greeting)s %(greeting.name)s%(greeting.suffix)s

<div class="alert alert-info"><strong>Note:</strong> sections are a <em>goconfig</em> feature.</div>

#### Arguments

Messages support one or more arguments. Arguments in messages are resolved using the same rules as the go `fmt` package. For example:

    greeting.name_arg=Hello %s!
    
***

## Resolving the client locale

In order to figure out which locale the user prefers Revel will look for a usable locale in the following places:

1. Language cookie

    Each request the framework will look for a cookie with the name defined in the application configuration (`i18n.cookie`). When such a cookie is found its value is 
    assumed to be the current locale. It's possible for the application to set this cookie's value in order to *force* the current locale.

2. Accept-Language HTTP header

    Revel will automatically parse the *Accept-Language HTTP header* for each incoming request. Each of the locales in the Accept-Language header value is evaluated 
    and stored - in order of qualification according to the [HTTP specification](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.4) - in the current 
    Revel `Request` instance. This information is later used by the various message resolving functions to determine the current locale.

3. Default language

    When all of the look-up methods above have returned no usable client locale, the framework will use the default locale as defined in the application configuration
    file (`i18n.default_language`).

### Retrieving the current locale

The application code can access the current locale from within a `Controller` using the `Controller.Args` map with the key `currentLocale`. For example:

<pre class="prettyprint lang-go">
func (c Application) Index() rev.Result {
	currentLanguage := c.Args["currentLocale"].(string)
	c.Render(currentLanguage)
}
</pre>

From a template, the current language can be retrieved from the current `renderArgs` instance. For example:

    <p>Current preferred language: {{.currentLocale}}</p>

***

## Resolving messages

Messages can be resolved from either a *view template* or a *controller*.

* Controller

    Each controller has a `Message(...)` function that can be used to resolve messages:

    ...

* Template

    ...
