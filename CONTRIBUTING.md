## Contribute to Revel

This describes how developers may contribute to Revel.

## Mission

Revel's mission is to provide a batteries-included framework for making large
scale web application development as efficient and maintainable as possible.

The design should be configurable and modular so that it can grow with the
developer. However, it should provide a wonderful un-boxing experience and
default configuration that can woo new developers and make simple web apps
straightforward. The framework should have an opinion about how to do all of the
common tasks in web development to reduce unnecessary cognitive load.

## How to Contribute

Presently there are no versioning or compatibility guarantees in place, so the
contribution process is not very formal.

### Discuss your idea

For the greatest chance of success, start with an email to
[revel-framework@googlegroups.com](mailto:revel-framework@googlegroups.com) to
discuss your contribution idea and design.

### How to fork (without breaking Go import paths)

Go uses the repository URL to import packages, so forking and go-getting the
forked project **will not work**.

Instead, this is the recommended way:

1. Fork Revel project on Github
2. In your clone of github.com/robfig/revel, add your fork as a remote.
3. Push to your fork to prepare a pull request.

Here is the command line: 
```
$ cd $GOPATH/src/github.com/robfig/revel              # Change directory to revel repo
$ git remote add fork git@github.com:$USER/revel.git  # Add your fork as a remote
$ git push fork master                                # After new commits, push to your fork
$ git pull origin master                              # Optionally, merge new changes from upstream
```

### Gofmt your code

Set your editor to run "go fmt" every time you save so that whitespace / style
comments are kept to a minimum.

Howtos:
* [Emacs](http://blog.golang.org/2013/01/go-fmt-your-code.html)

### Write a test (and maybe a benchmark)

Significant new features require tests. Besides unit tests, it is also possible
to test a feature by exercising it in one of the sample apps and verifying its
operation using that app's test suite. This has the added benefit of providing
example code for developers to refer to.

Benchmarks are helpful but not required.

### Run the tests

Typically running the main set of unit tests will be sufficient:

	$ go test github.com/robfig/revel

Refer to the
[Travis configuration](https://github.com/robfig/revel/blob/master/.travis.yml)
for the full set of tests.  They take less than a minute to run.

### Document your feature

The [Revel web site](http://robfig.github.io/revel/) is hosted on Github-pages and 
[built with Jekyll](https://help.github.com/articles/using-jekyll-with-pages).

To develop the site locally:

	# Clone a second repository and check out the branch
	$ git clone git@github.com:robfig/revel.git
	$ cd revel
	$ git checkout gh-pages

	# Install / run Jekyll 1.0.3 to generate the site, and serve the result
	$ gem install jekyll -v 1.0.3
	$ jekyll build --watch --safe -d test/revel &
	$ cd test
	$ python -m SimpleHTTPServer 8088

	# Now load in your browser
	$ open http://localhost:8088/revel

Any changes you make to the site should be reflected within a few seconds.

## Potential Projects

These are outstanding feature requests, roughly ordered by priority.
Additionally, there are frequently smaller feature requests or items in the
[issues](https://github.com/robfig/revel/issues?labels=contributor+ready&page=1&state=open).

1.  Better ORM support.  Provide more samples (or modules) and better documentation for setting up common situations like SQL database, Mongo, LevelDB, etc.
2.	Support for other templating languages (e.g. mustache, HAML).  Make TemplateLoader pluggable.  Use Pongo instead of vanilla Go templates (and update the samples)
12.	Test Fixtures
13.	Authenticity tokens for CSRF protection
5. Coffeescript pre-processor.  Could potentially use [otto](https://github.com/robertkrimen/otto) as a native Go method to compiling.
6.  SCSS/LESS pre-processor.
4.	GAE support.  Some progress made in the 'appengine' branch -- the remaining piece is running the appengine services in development.
3.  More Form helpers (template funcs).
5.	A Mongo module (perhaps with a sample app)
9.	Easy emailer support (e.g. to email exception logs to developer, or even to email users),
9.  Deployment to OpenShift (support, documentation, etc)
16.	Improve the logging situation.  The configuration is a little awkward and not very powerful.  Integrating something more powerful would be good. (like [seelog](https://github.com/cihub/seelog) or [log4go](https://code.google.com/p/log4go/))
11.	ETags, cache controls
14.	A module or plugins for adding HTTP Basic Auth
7.	Allowing the app to hook into the source code processing step
