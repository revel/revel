package binding

import (
	"mime/multipart"
	"os"
	"reflect"
)

var (
	fileBinder = Binder{bindFile, nil, purgeFiles}
	tmpFiles *[]os.File
)


func bindFile(files *map[string][]*multipart.FileHeader, name string, typ reflect.Type) reflect.Value {
	reader := getMultipartFile(files, name)
	if reader == nil {
		return reflect.Zero(typ)
	}

	// If it's already stored in a temp file, just return that.
	if osFile, ok := reader.(*os.File); ok {
		return reflect.ValueOf(osFile)
	}

	// Otherwise, have to store it.
	tmpFile, err := ioutil.TempFile("", "revel-upload")
	if err != nil {
		// TODO WARN.Println("Failed to create a temp file to store upload:", err)
		return reflect.Zero(typ)
	}

	// Register it to be deleted after the request is done.
	tmpFiles = append(tmpFiles, tmpFile)

	_, err = io.Copy(tmpFile, reader)
	if err != nil {
		// TODO WARN.Println("Failed to copy upload to temp file:", err)
		return reflect.Zero(typ)
	}

	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		// TODO WARN.Println("Failed to seek to beginning of temp file:", err)
		return reflect.Zero(typ)
	}

	return reflect.ValueOf(tmpFile)
}


// Helper that returns an upload of the given name, or nil.
func getMultipartFile(files *map[string][]*multipart.FileHeader, name string) multipart.File {
	for _, fileHeader := range files[name] {
		file, err := fileHeader.Open()
		if err == nil {
			return file
		}
		// TODO WARN.Println("Failed to open uploaded file", name, ":", err)
	}
	return nil
}


func purgeFiles() (err error) {
	for _, tmpFile := range c.Params.tmpFiles {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			return
		}
	}
}

func init() {
	TypeBinders[reflect.TypeOf(&os.File{})] = fileBinder
}
