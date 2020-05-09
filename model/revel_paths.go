package model

type  RevelPaths      struct {
	Import    string
	Source    string
	Base      string
	Code      []string              // Consolidated code paths
	Template  []string              // Consolidated template paths
	Config    []string              // Consolidated configuration paths
	ModuleMap map[string]*RevelUnit // The module path map
}
