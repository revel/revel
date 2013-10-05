package controllers

import "github.com/robfig/revel"

func init() {
	revel.ActionRestrictions = []revel.ActionRestriction{
		{"Application", "Index", AskToLogIn},
		{"Hotels", "*", MustBeLoggedIn},
	}
	revel.OnAppStart(Init)
	revel.InterceptMethod((*GorpController).Begin, revel.BEFORE)
	revel.InterceptMethod(Application.AddUser, revel.BEFORE)
	revel.InterceptMethod((*GorpController).Commit, revel.AFTER)
	revel.InterceptMethod((*GorpController).Rollback, revel.FINALLY)
}
