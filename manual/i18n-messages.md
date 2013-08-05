---
title: Messages
layout: manual
---

Messages are used to externalize pieces of text in order to be able to provide translations for them. Revel
supports message files organized per language, automatic locale look-up, cookie-based overrides and message 
nesting and arguments.

#### Glossary
* Locale: a combination of *language* and *region* that indicates a user language preference, eg. `en-US`.
* Language: the language part of a locale, eg. `en`. Language identifiers are expected to be [ISO 639-1 codes](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Region: the region part of a locale, eg. `US`. Region identifiers are expected to be [ISO 3166-1 alpha-2 codes](http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2).

## Sample application

The way Revel handles message files and internationalization in general is similar to most other web frameworks out there. For those of you that wish to get
started straight away without going through the specifics, there is a sample application `revel/samples/i18n` that you can have a look at which demonstrates 
all the basics.

## Message files

Messages are defined in message files. These files contain the actual text that will be used while rendering the view (or elsewhere in your application if you so desire). 
When creating new message files, there are a couple of rules to keep in mind:

* All message files should be stored in the `messages` directory in the application root.
* The file extension determines the *language* of the message file and should be an [ISO 639-1 code](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).
* Message files should be UTF-8 encoded. While this is not a hard requirement, it is best practice.
* Each message file is effectively a [goconfig file](https://github.com/robfig/config) and supports all goconfig features.

### Organizing message files

There are no restrictions on message file names; a message file name can be anything as long as it has a valid extention. There is also no restriction on the *amount*
of files per language. When the application starts, Revel will parse all message files with a valid extension in the `messages` directory and merge them according to their 
language. This means that you are free to organize the message files however you want.

For example, you may want to take a traditional approach and define 1 single message file per language:

    /app
        /messages
            messages.en
            messages.fr
            ...

Another approach would be to create *multiple files* for the *same language* and organize them based on the kind of messages they contain:

    /app
        /messages
            labels.en
            warnings.en
            labels.fr
            warnings.fr
            ...

<div class="alert alert-block"><strong>Important note:</strong> while it's technically possible to define the same <em>message key</em> in multiple files with the same language, this will result in unpredictable behaviour. When using multiple files per language, take care to keep your message keys unique so that keys will not be overwritten after the files are merged!</div>

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
[Regions](#regions) for more information).

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

Messages in message files can reference other messages. This allows users to compose a single message from one or more other messages. The syntax for referencing other messages 
is `%(key)s`. For example:

    greeting=Hello 
    greeting.name=Rob
    greeting.suffix=, welcome to Revel!
    greeting.full=%(greeting)s %(greeting.name)s%(greeting.suffix)s

<div class="alert alert-info"> 
    <p><strong>Notes:</strong></p>
    <ul>
        <li>Referencing is a <em>goconfig</em> feature.</li>
        <li>Because message files are merged, it's perfectly possible to reference messages in other files provided they are defined for the same language.</li>
    </ul>
</div>

#### Arguments

Messages support one or more arguments. Arguments in messages are resolved using the same rules as the go `fmt` package. For example:

    greeting.name_arg=Hello %s!

Arguments are resolved in the same order as they are given, see [Resolving messages](#resolving_messages).

## Resolving the client locale

In order to figure out which locale the user prefers Revel will look for a usable locale in the following places:

1. Language cookie

    Each request the framework will look for a cookie with the name defined in the application configuration (`i18n.cookie`). When such a cookie is found its value is 
    assumed to be the current locale. All other resolution methods will be skipped when a cookie has been found.

2. `Accept-Language` HTTP header

    Revel will automatically parse the `Accept-Language` HTTP header for each incoming request. Each of the locales in the `Accept-Language` header value is evaluated 
    and stored - in order of qualification according to the [HTTP specification](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.4) - in the current 
    Revel `Request` instance. This information is later used by the various message resolving functions to determine the current locale.

    For more information see [Parsed Accept-Language HTTP header](#parsed_acceptlanguage_http_header).

3. Default language

    When all of the look-up methods above have returned no usable client locale, the framework will use the default language as defined in the application configuration
    file (`i18n.default_language`).

When the requested message could not be resolved at all, a specially formatted string containing the original message is returned.

<div class="alert alert-info"><strong>Note:</strong> the <code>Accept-Language</code> header is <strong>always</strong> parsed and stored in the current <code>Request</code>, even when a language cookie has been found. In such a case, the values from the header are simply never used by the message resolution functions, but they're still available to the application in case it needs them.</div>

### Retrieving the current locale

The application code can access the current locale from within a `Request` using the `Request.Locale` property. For example:

<pre class="prettyprint lang-go">
func (c App) Index() revel.Result {
	currentLocale := c.Request.Locale
	c.Render(currentLocale)
}
</pre>

From a template, the current locale can be retrieved from the `currentLocale` property from the current `renderArgs`. For example:

<pre class="prettyprint lang-html">
    &#x3c;p&#x3e;Current preferred locale: &#x7b;&#x7b;.currentLocale&#x7d;&#x7d;&#x3c;/p&#x3e;
</pre>

### Parsed Accept-Language HTTP header

In case the application needs access to the `Accept-Language` HTTP header for the current request it can retrieve it from the `Request` instance of the `Controller`. The `AcceptLanguages` field 
- which is a slice of `AcceptLanguage` instances - contains all parsed values from the respective header, sorted per qualification with the most qualified values first in the slice. For example:

<pre class="prettyprint lang-go">
func (c App) Index() revel.Result {
    // Get the string representation of all parsed accept languages
    c.RenderArgs["acceptLanguageHeaderParsed"] = c.Request.AcceptLanguages.String()
    // Returns the most qualified AcceptLanguage instance
    c.RenderArgs["acceptLanguageHeaderMostQualified"] = c.Request.AcceptLanguages[0]

    c.Render()
}
</pre>

For more information see the [HTTP specification](http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.4).

## Resolving messages

Messages can be resolved from either a *controller* or a *view template*.

### Controller

Each controller has a `Message(message string, args ...interface{})` function that can be used to resolve messages using the current locale. For example:

<pre class="prettyprint lang-go">
func (c App) Index() revel.Result {
	c.RenderArgs["controllerGreeting"] = c.Message("greeting")
	c.Render()
}
</pre>

### Template

To resolve messages using the current locale from templates there is a *template function* `msg` that you can use. For example:

<pre class="prettyprint lang-html">
    &#x3c;p&#x3e;Greetings without arguments: &#x7b;&#x7b;msg . "greeting"&#x7d;&#x7d;&#x3c;/p&#x3e;
    &#x3c;p&#x3e;Greetings: &#x7b;&#x7b;msg . "greeting.full.name" "Tommy Lee Jones"&#x7d;&#x7d;&#x3c;/p&#x3e;
</pre>

<div class="alert alert-info"><strong>Note:</strong> the signature of the <code>msg</code> function is <code>msg . "message name" "argument" "argument"</code>. If there are no arguments, simply do not include any.</div>

## Configuration

<table class="table table-striped">
    <thead>
        <tr>
            <th style="width: 15%">File</th>
            <th style="width: 25%">Option</th>
            <th style="width: 60%">Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td>
                <code>app.conf</code>
            </td>
            <td>
                <code>i18n.cookie</code>
            </td>
            <td>
                The name of the language cookie. Should always be prefixed with the Revel cookie prefix to avoid cookie name conflicts.
            </td>
        </tr>
        <tr>
            <td>
                <code>app.conf</code>
            </td>
            <td>
                <code>i18n.default_language</code>
            </td>
            <td>
                The default locale to use in case no preferred locale could be found.
            </td>
        </tr>
    </tbody>
</table>
