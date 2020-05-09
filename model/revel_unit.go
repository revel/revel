package model

const (
	APP    RevelUnitType = 1 // App always overrides all
	MODULE RevelUnitType = 2 // Module is next
	REVEL  RevelUnitType = 3 // Revel is last
)

type (
	RevelUnit struct {
		Name       string        // The friendly name for the unit
		Config     string        // The config file contents
		Type       RevelUnitType // The type of the unit
		Messages   string        // The messages
		BasePath   string        // The filesystem path of the unit
		ImportPath string        // The import path for the package
		Container  *RevelContainer
	}
	RevelUnitList []*RevelUnit
	RevelUnitType int

)
