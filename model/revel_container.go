package model


// The single instance object that has the config populated to it
type (
	RevelContainer struct {
		Controller RevelController
		Paths RevelPaths
}
)