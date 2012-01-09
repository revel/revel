# Go Play!

This is a port of the amazing [Play! framework](http://www.playframework.org) to Go.

It is nowhere near ready for anyone to look at, much less use.

# To try it out:
- clone this repo into your GOPATH.  e.g:

```
export GOPATH=/Users/$USER/gocode/src
mkdir -p $GOPATH
cd $GOPATH
git clone github.com/robfig/go-play/play
```

- install [gb](http://code.google.com/p/go-gb/): `goinstall github.com/skelterjohn/go-gb/gb`
- build the play command line tool with gb: `gb src/play/cmd`
- run the sample app: `./bin/play`

# Scratch space

## Five cool things you can do.

# Routing

- routes file like before.

# Controllers

```go
type Controller struct {

  // Per-request
  Params
  Args
  Action
  Method
}
```


A typical method:

```go

myapp/controllers/users.go:

type Users struct {
  *play.Controller
}

func (*Users u) showUser(userId int) play.Result {

  // Look for myapp/views/Users/showUser.*
  // If it ends in .mustache, use that.
  // If it ends in .html .. default to something?
  return u.Render(map[string]User{"user": user})  // Template uses map
  // or
  return u.Render(user)  // Template uses fields of user.
}

type User struct { .. }

func (*Users u) saveUser(userId int) play.Result r {
  user := u.ParseParams(User)  // Can you pass type?
  // or
  id := play.Param("id", int)
  // or
  user := User{
    id: play.Param("id", int),
  }
}

```

# Templates

- Able to use Mustache server and client side sharing the same template.
- Able to use more powerful Go templates if not needing client side.
- Come with bootstrap (e.g. demo page uses)
- Support coffeescript/less compilation ?  (need a story for it)


# Development

- In development, a helper Go server proxies requests to real Go server.  It recompiles / restarts  when necessary.


# Work plan

1. Get simple server working: A route from routes, a controller, a no-arg mustache view
2. Get hot-compile working: A go proxy, compile, show compile errors

Routing
ORM
Http/Request/Response/Session/Flash
- Gorilla sessions, except its session is like a Play cache.
Form validation
Data binding
- Start with Gorilla schema package
Templating
Interceptors
Libraries (XML, File IO, WS, OAuth, Email, Images)
Async programming (suspend, resume)
Websockets
Internationalization
Jobs
Plugins


server.go:
- Router
- TemplateLoader

mvc.go:


tmpl.go:
How to test?


Is it necessary to recompile/restart the whole server when e.g. a Controller changes?

harness
- User builds and runs harness, passing path to play app.
- Harness loads the app's config, builds it.
- Harness starts it.


Todo:
- Build the user app correctly (ie, with dependency analysis, using godg)
 - See about goinstall godag .. making a makefile is too hard.
- Show go compile errors prettily.
- Interceptors
- Reverse routing
- application.conf parsing
- Gorilla schema / support field[0].property
- Cookies
- Flash
- Form validation (Gorilla schema going to do it?)
- Jobs
- Plugins


// @Before controllers.Login

// Register interceptors.
func init() {
	play.Intercept((*Application) nil, )
}
