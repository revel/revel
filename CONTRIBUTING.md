## Contributing to Revel

This describes how developers may contribute to Revel.

## Mission

Revel's mission is to provide a batteries-included framework for making large
scale web application development as efficient and maintainable as possible.

The design should be configurable and modular so that it can grow with the
developer. However, it should provide a wonderful un-boxing experience and
default configuration that can woo new developers and make simple web apps
straight forward. The framework should have an opinion about how to do all of the
common tasks in web development to reduce unnecessary cognitive load.

Perhaps most important of all, Revel should be a joy to use. We want to reduce
the time spent on tedious boilerplate functionality and increase the time
available for creating polished solutions for your application's target users.

## How to Contribute

### Join the Community

The first step to making Revel better is joining the community! You can find the
community on:

* [Google Groups](https://groups.google.com/forum/#!forum/revel-framework) via [revel-framework@googlegroups.com](mailto:revel-framework@googlegroups.com)
* [GitHub Issues](https://github.com/revel/revel/issues)
* [StackOverflow Questions](http://stackoverflow.com/questions/tagged/revel)
* [IRC](http://webchat.freenode.net/?channels=%23revel&uio=d4) via #revel on Freenode

Once you've joined, there are many ways to contribute to Revel:

* Report bugs (via GitHub)
* Answer questions of other community members (via Google Groups or IRC)
* Give feedback on new feature discussions (via GitHub and Google Groups)
* Propose your own ideas (via Google Groups or GitHub)

### How Revel is Developed

We have begun to formalize the development process by adopting pragmatic
practices such as:

* Developing on the `develop` branch
* Merging `develop` branch to `master` branch in 6 week iterations
* Tagging releases with MAJOR.MINOR syntax (e.g. v0.8)
** We may also tag MAJOR.MINOR.HOTFIX releases as needed (e.g. v0.8.1) to
address urgent bugs. Such releases will not introduce or change functionality
* Managing bugs, enhancements, features and release milestones via GitHub's Issue Tracker
* Using feature branches to create pull requests
* Discussing new features **before** hacking away at it


### How to Correctly Fork

Go uses the repository URL to import packages, so forking and `go get`ing the
forked project **will not work**.

Instead, follow these steps:

1. Install Revel normally
2. Fork Revel on GitHub
3. Add your fork as a git remote

Here's the commands to do so:
```
$ go get github.com/revel/revel                        # Install Revel
$ cd $GOPATH/src/github.com/revel/revel                # Change directory to revel repo
$ git remote add fork git@github.com:$USER/revel.git  # Add your fork as a remote, where $USER is your GitHub username
```

### Create a Feature Branch & Code Away!

Now that you've properly installed and forked Revel, you are ready to start coding (assuming
you have a validated your ideas with other community members)!

In order to have your pull requests accepted, we recommend you make your changes to Revel on a
new git branch. For example,
```
$ git checkout -b feature/useful-new-thing origin/develop    # Create a new branch based on develop and switch to it
$ ...                                                        # Make your changes and commit them
$ git push fork feature/useful-new-thing                     # After new commits, push to your fork
```

### Format Your Code

Remember to run `go fmt` before committing your changes.
Many Go developers opt to have their editor run `go fmt` automatically when
saving Go files.

Additionally, follow the [core Go style conventions](https://code.google.com/p/go-wiki/wiki/CodeReviewComments)
to have your pull requests accepted.

### Write Tests (and Benchmarks for Bonus Points)

Significant new features require tests. Besides unit tests, it is also possible
to test a feature by exercising it in one of the sample apps and verifying its
operation using that app's test suite. This has the added benefit of providing
example code for developers to refer to.

Benchmarks are helpful but not required.

### Run the Tests

Typically running the main set of unit tests will be sufficient:

```
$ go test github.com/revel/revel
```

Refer to the
[Travis configuration](https://github.com/revel/revel/blob/master/.travis.yml)
for the full set of tests.  They take less than a minute to run.

### Document Your Feature

Due to the wide audience and shared nature of Revel, documentation is an essential
addition to your new code. **Pull requests risk not being accepted** until proper
documentation is created to detail how to make use of new functionality.

The [Revel web site](http://revel.github.io/) is hosted on GitHub Pages and
[built with Jekyll](https://help.github.com/articles/using-jekyll-with-pages).

To develop the Jekyll site locally:

    # Clone the documentation repository
    $ git clone git@github.com:revel/revel.github.io
    $ cd revel.github.io

    # Install and run Jekyll to generate and serve the site
    $ gem install jekyll kramdown
    $ jekyll serve --watch

    # Now load in your browser
    $ open http://localhost:4000/

Any changes you make to the site should be reflected within a few seconds.

### Submit Pull Request

Once you've done all of the above & pushed your changes to your fork, you can create a pull request for review and acceptance.

## Potential Projects

These are outstanding feature requests, roughly ordered by priority.
Additionally, there are frequently smaller feature requests or items in the
[issues](https://github.com/revel/revel/issues?labels=contributor+ready&page=1&state=open).

1.	Better ORM support.  Provide more samples (or modules) and better documentation for setting up common situations like SQL database, Mongo, LevelDB, etc.
1.	Support for other templating languages (e.g. mustache, HAML).  Make TemplateLoader pluggable.  Use Pongo instead of vanilla Go templates (and update the samples)
1.	Test Fixtures
1.	Authenticity tokens for CSRF protection
1.	Coffeescript pre-processor.  Could potentially use [otto](https://github.com/robertkrimen/otto) as a native Go method to compiling.
1.	SCSS/LESS pre-processor.
1.	GAE support.  Some progress made in the 'appengine' branch -- the remaining piece is running the appengine services in development.
1.	More Form helpers (template funcs).
1.	A Mongo module (perhaps with a sample app)
1.	Deployment to OpenShift (support, documentation, etc)
1.	Improve the logging situation.  The configuration is a little awkward and not very powerful.  Integrating something more powerful would be good. (like [seelog](https://github.com/cihub/seelog) or [log4go](https://code.google.com/p/log4go/))
1.	ETags, cache controls
1.	A module or plugins for adding HTTP Basic Auth
1.	Allowing the app to hook into the source code processing step
