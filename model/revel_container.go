package model

import "github.com/revel/revel/utils"

// The single instance object that has the config populated to it
type RevelContainer struct {
	Controller struct {
		Reuse              bool                              // True if the controllers are reused Set via revel.controller.reuse
		Stack              *utils.SimpleLockStack            // size set by revel.controller.stack,  revel.controller.maxstack
		CachedMap          map[string]*utils.SimpleLockStack // The map of reusable controllers
		CachedStackSize    int                               // The default size of each stack in CachedMap Set via revel.cache.controller.stack
		CachedStackMaxSize int                               // The max size of each stack in CachedMap Set via revel.cache.controller.maxstack
	}
}
