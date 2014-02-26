package controllers

import (
	"github.com/revel/revel"
	"net/http"
	"net/http/pprof"
)

type Pprof struct {
	*revel.Controller
}

// The PprofHandler type makes it easy to call the net/http/pprof handler methods
// since they all have the same method signature
type PprofHandler func(http.ResponseWriter, *http.Request)

func (r PprofHandler) Apply(req *revel.Request, resp *revel.Response) {
	r(resp.Out, req.Request)
}

func (c Pprof) Profile() revel.Result {
	return PprofHandler(pprof.Profile)
}

func (c Pprof) Symbol() revel.Result {
	return PprofHandler(pprof.Symbol)
}

func (c Pprof) Cmdline() revel.Result {
	return PprofHandler(pprof.Cmdline)
}

func (c Pprof) Index() revel.Result {
	return PprofHandler(pprof.Index)
}
