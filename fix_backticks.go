package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	path := "internal/repository/binary_file_repository.go"
	data, err := ioutil.ReadFile(path)
	if err != nil { panic(err) }
	
	content := string(data)
	content = strings.ReplaceAll(content, "query := \n\t\tSELECT", "query := \x60\n\t\tSELECT")
	content = strings.ReplaceAll(content, "\t\n\trows, err", "\t\x60\n\trows, err")
	content = strings.ReplaceAll(content, "\t\n\t\n\trows, err", "\t\x60\n\trows, err")
	
	// Also fix the first one which didn't have \t
	content = strings.ReplaceAll(content, "query := \n\t\t\tCOALESCE", "query := \x60\n\t\tSELECT\n\t\t\tCOALESCE")
	
	ioutil.WriteFile(path, []byte(content), 0644)
}
