---
title: Messages
layout: manual
---

Messages are used to externalize pieces of text in order to be able to provide translations for them. Revel
supports message files organized per locale (a combination of *language* and *region*), transparent locale 
look-up, cookie-based overrides and message nesting and arguments.

### Glossary

Throughout this manual, the following terms will be used frequently:

* Locale
  A combination of *language* and *region* that indicates a user language preference, eg. `en-US`.

* Language
  The language part of a locale, eg. `en`. Language identifiers are expected to be [ISO 639-1 codes](http://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).

* Region
  The region part of a locale, eg. `US`. Region identifiers are expected to be [ISO 3166-1 alpha2 codes](http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2).
