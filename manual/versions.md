---
title: Versioning
layout: manual
---

A great deal has been written about Go's package versioning situation (by
@nathany in particular). However, at this time, there is no community standard
for managing package versions.  Therefore, it is left up to the end developer to
ensure their software builds safely and reproducibly.

If you use Revel for a production application, it is *your* responsibility to
avoid breakages due to incompatible changes.  Your build process should not
involve "go get"ing the master branch of Revel.

The simplest way to handle this is to check the code for Revel and all
dependencies into your repository.  If you use git, they can be embedded
efficiently as sub-repos.

Alternatively, try one of the package managers described in the linked article.

* [Go package versioning](http://nathany.com/go-packages/)

