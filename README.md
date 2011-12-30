# Go Play!

This is a port of the amazing [Play! framework][http://www.playframework.org] to Go.

It is nowhere near ready for anyone to look at, much less use.

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
  play.Controller
}

func (*Users u) showUser(userId int) play.Result r {

  // Look for myapp/views/Users/showUser.*
  // If it ends in .mustache, use that.
  // If it ends in .html .. default to something?
  u.Render(map[string]User{"user": user})  // Template uses map as usual
  // or
  u.Render(user)  // Template uses fields of user.
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
