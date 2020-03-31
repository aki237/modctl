package main

import (
	"io/ioutil"

	"golang.org/x/mod/modfile"
)

func loadModFile() (*modfile.File, error) {
	bs, err := ioutil.ReadFile("go.mod")
	if err != nil {
		return nil, err
	}

	return modfile.Parse("go.mod", bs, nil)
}
