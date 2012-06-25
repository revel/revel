---
title: How Revel Works
layout: manual
---

- The command line tool runs a harness that acts as a reverse proxy.
- It listens on port 9000 and watches the app files for changes.
- It forwards requests to the running server.  If the server isn't running or a source file has changed since the last request, it rebuilds the app.
- If it needs to rebuild the app, the harness analyzes the source code and produces a `app/tmp/main.go` file that contains all of the meta information necessary required to support the various magic as well as runs the real app server.
- It uses `go build` to compile the app.  If there is a compile error, it shows a helpful error page to the user as the response.
- If the app compiled successfully, it runs the app and forwards the request when it detects that the app server has finished starting up.

