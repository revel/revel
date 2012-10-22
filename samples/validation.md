---
title: Validation
layout: samples
---

The validation app demonstrates every way that the Validation system may be used
to good effect.

Here are the contents of the app:

	validation/app/
		models
			user.go     # User struct and validation routine.
		controllers
			app.go      # Introduction
			sample1.go  # Validating simple fields with error messages shown at top of page.
			sample2.go  # Validating simple fields with error messages shown inline.
			sample3.go  # Validating a struct with error messages shown inline.

[Browse the code on Github](https://github.com/robfig/revel/tree/master/samples/validation)
