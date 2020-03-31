package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/mod/modfile"
)

type Upgrader struct {
	Packages map[string]*Version
	file     *modfile.File
}

type Package struct {
	Path    string
	Version *Version
}

func (p *Package) Import() string {
	if p.Version.Major <= 1 {
		return p.Path
	}

	return fmt.Sprintf("%s/v%d", p.Path, p.Version.Major)
}

func _package(path string, v *Version) *Package {
	return &Package{Path: path, Version: v}
}

type Version struct {
	Major        int
	Minor        int
	Patch        int
	Suffix       string
	Incompatible bool
}

func (v *Version) String() string {
	suffix := "-" + v.Suffix
	if suffix == "-" {
		suffix = ""
	}
	return fmt.Sprintf("v%d.%d.%d%s", v.Major, v.Minor, v.Patch, suffix)
}

func parseVersion(x string) (*Version, error) {
	isIncompatible := strings.HasSuffix(x, "+incompatible")
	if isIncompatible {
		x = strings.TrimSuffix(x, "+incompatible")
	}
	splits := strings.SplitN(x, ".", 3)
	if len(splits) != 3 {
		return nil, errors.New("Invalid version string: " + x)
	}
	major, err := strconv.Atoi(strings.TrimPrefix(splits[0], "v"))
	if err != nil {
		return nil, err
	}

	minor, err := strconv.Atoi(splits[1])
	if err != nil {
		return nil, err
	}

	suffix := ""

	patchSplit := strings.SplitN(splits[2], "-", 2)
	if len(patchSplit) > 1 {
		suffix = patchSplit[1]
	}

	patch, err := strconv.Atoi(patchSplit[0])
	if err != nil {
		return nil, err
	}

	return &Version{
		Major:  major,
		Minor:  minor,
		Patch:  patch,
		Suffix: suffix,
	}, nil
}

func (u *Upgrader) fixer(path, version string) (string, error) {
	return version, nil
}

func (u *Upgrader) Exec(ctx *cli.Context) error {
	u.Packages = make(map[string]*Version)
	args := ctx.Args().Slice()
	if len(args) < 1 {
		return errors.New("required package with a revision")
	}

	var err error
	for _, pkg := range args {
		splits := strings.Split(pkg, "@")
		if len(splits) != 2 {
			return errors.New("invalid package import")
		}

		u.Packages[splits[0]], err = parseVersion(splits[1])
		if err != nil {
			return err
		}
	}

	bs, err := ioutil.ReadFile("go.mod")
	if err != nil {
		return err
	}

	x, err := modfile.Parse("go.mod", bs, u.fixer)
	if err != nil {
		return err
	}
	u.file = x

	for _, req := range u.file.Require {
		if err := u.Analyze(req); err != nil {
			return err
		}
	}

	return nil
}

func (u *Upgrader) Analyze(req *modfile.Require) error {
	pkgpath := req.Mod.Path
	version := req.Mod.Version
	inv, err := parseVersion(version)
	if err != nil {
		return err
	}
	pv := strings.TrimSuffix(pkgpath, fmt.Sprintf("v%d", inv.Major))

	rv, ok := u.Packages[pv]
	if !ok {
		return nil
	}
	if inv.Major == rv.Major {
		return nil
	}
	newImport := fmt.Sprintf("%s/v%d", pv, rv.Major)
	fmt.Printf("Incrementing major version of %s from %s to %s\n", pkgpath, version, rv)
	fmt.Printf("Changing import from %s to %s\n", pv, newImport)
	u.file.DropRequire(pkgpath)
	u.file.AddNewRequire(newImport, rv.String(), false)
	content, err := u.file.Format()
	if err != nil {
		return err
	}
	gofiles := make([]string, 0)

	// Fixup package imports.
	err = filepath.Walk(".", func(pathName string, info os.FileInfo, err error) error {
		if strings.HasSuffix(pathName, ".go") {
			gofiles = append(gofiles, pathName)
		}

		return err
	})
	if err != nil {
		return err
	}

	for _, file := range gofiles {
		err := replaceImport(file, _package(pv, inv), _package(pv, rv))
		if err != nil {
			return err
		}
	}

	// write content to go.mod.
	return ioutil.WriteFile("go.mod", content, 0644)
}

func e(cmd string) {
	c := exec.Command("bash", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr
	c.Run()
}

func _rq(str string) string {
	return str + "\""
}

func _lq(str string) string {
	return "\"" + str
}

func replaceImport(file string, oldImport *Package, newImport *Package) error {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	_hp := strings.HasPrefix
	// Print the imports from the file's AST.
	for _, s := range f.Imports {
		if _hp(s.Path.Value, _lq(oldImport.Import())) &&
			!_hp(s.Path.Value, _lq(newImport.Import())) {
			s.Path.Value = strings.Replace(s.Path.Value, oldImport.Import(), newImport.Import(), 1)
		}
	}

	b := bytes.NewBuffer(nil)
	err = format.Node(b, fset, f)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(file, b.Bytes(), 0644)
}
