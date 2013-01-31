## Contribute to Revel

This describes how developers may contribute to Revel.

## Mission

Revel's mission is to provide a batteries-included framework for making large
scale web application development as efficient and maintainable as possible.

The Play! Framework had a huge impact on ability of the author to deliver
software of "business value" extremely rapidly.  Bringing a similar toolkit and
API to the Go language could provide a framework that delivers both productivity
*and* efficiency of computing resources.  (Go programs are generally quite
efficient by Java or Scala standards in terms of both memory and CPU.)  Plus, Go
subjectively is a much more pleasant and scalable language to develop in.

## How to Contribute

The ideal process for a successful contribution looks like this:

1. Send email to revel-framework@googlegroups.com with your idea.
2. Within 24 hours (usually), @robfig will respond with a "yes", "no", or discussion.
3. Upon "yes", fork the repository, and prepare + send a Pull Request
4. (Optional) If your change affects the developer-facing functionality, it is appreciated (but not mandatory) to add it to the manual.  Switch to the gh-pages branch of the repository, document your change, and send a Pull Request for that as well.
5. @robfig will provide a code review, and when no outstanding comments are left he will merge the pull request(s).

In other words, not much red tape.

## Potential Projects

These are outstanding feature requests, roughly ordered by priority.

1.  Better ORM support.  (Investigate Hood)
1.	Running jobs on a schedule (like the Play! Jobs framework)
2.	Support for other templating languages (e.g. mustache, HAML)
6.	Better reverse routing (the current thing sucks, the stuff Play has rocks)
4.	GAE support (some progress made in the gae branch, just lacking running the appengine services in development)
3.  Form helpers in general.
5. SCSS/LESS/Coffeescript pre-processors.
5.	A Mongo module (perhaps with a sample app)
8.	A Cache interface (a la Play)
9.	Easy emailer support (e.g. to email exception logs to developer, or even to email users),
10.	Cross-compilation in the "package" command. (Generally, working on easy deployment or having a full article on deploying to Heroku)
11.	ETags, cache controls
12.	Test Fixtures
3.	Performance tests/tuning
13.	Authenticity tokens for CSRF protection
14.	A module or plugins for adding HTTP Basic Auth
7.	Allowing the app to hook into the source code processing step
15.	More tests for revel code
16.	Improve the logging situation (the configuration is a little awkward, and it would be nice to have a shorter name to log with than "rev.ERROR.Println")
