package revel

import (
	"bytes"
	"regexp"
)

// Module matching template syntax allows for modules to replace this text with the name of the module declared on import
// this allows the reverse router to use correct syntax
// Match _LOCAL_.static or  _LOCAL_|
var namespaceReplacement = regexp.MustCompile(`(_LOCAL_)(\.(.*?))?\\`)

// Function to replace the bytes data that may match the _LOCAL_ namespace specifier,
// the replacement will be the current module.Name
func namespaceReplace(fileBytes []byte, module *Module) []byte {
	newBytes, lastIndex := &bytes.Buffer{}, 0
	matches := namespaceReplacement.FindAllSubmatchIndex(fileBytes, -1)
	for _, match := range matches {
		// Write up to first bytes
		newBytes.Write(fileBytes[lastIndex:match[0]])
		// skip ahead index to match[1]
		lastIndex = match[3]
		if match[4] > 0 {
			// This match includes the module name as imported by the module
			// We could transform the module name if it is different..
			// For now leave it the same
			// so _LOCAL_.static| becomes static|
			lastIndex++
		} else {
			// Inject the module name
			newBytes.Write([]byte(module.Name))
		}
	}
	// Write remainder of document
	newBytes.Write(fileBytes[lastIndex:])
	return newBytes.Bytes()
}
