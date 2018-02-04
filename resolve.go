package dots

import (
	"errors"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func isDir(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && fi.IsDir()
}

func isFile(filename string) bool {
	fi, err := os.Stat(filename)
	return err == nil && !fi.IsDir()
}

func readDir(dirname string, recurse bool) ([]string, error) {
	var files []string
	var appendFile = func(file string, info os.FileInfo, err error) error {
		if strings.HasSuffix(file, ".go") {
			files = append(files, file)
		}
		return nil
	}
	if recurse {
		err := filepath.Walk(dirname, appendFile)
		return files, err
	}
	res, err := ioutil.ReadDir(dirname)
	if err == nil {
		for _, f := range res {
			appendFile(f.Name(), f, nil)
		}
	}
	return files, err
}

func readPackage(packageName string) []string {
	pkg, _ := build.Import(packageName, ".", 0)
	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	files = append(files, pkg.TestGoFiles...)
	return files
}

func resolvePattern(pattern string) ([]string, error) {
	recurse := false
	if strings.HasSuffix(pattern, "/...") {
		recurse = true
		pattern = strings.Replace(pattern, "/...", "", 1)
	}
	if isDir(pattern) {
		return readDir(pattern, recurse)
	}
	if isFile(pattern) {
		return []string{pattern}, nil
	}
	return readPackage(pattern), nil
}

func resolvePatterns(patterns []string) ([]string, []error) {
	var paths []string
	var errs []error
	for _, s := range patterns {
		res, err := resolvePattern(s)
		if err != nil {
			errs = append(errs, errors.New(`unable to resolve "`+s+`": `+err.Error()))
		} else {
			paths = append(paths, res...)
		}
	}
	return paths, errs
}

// Resolve accepts a slice of paths with optional "..." placeholder and a slice with paths to be skipped.
// The final result is the set of all files from the selected directories subtracted with
// the files in the skip slice.
func Resolve(includePatterns, skipPatterns []string) ([]string, []error) {
	if len(includePatterns) == 0 {
		return []string{"."}, nil
	}

	skip, errs := resolvePatterns(skipPatterns)
	filter := newPathFilter(skip)

	pathSet := map[string]bool{}
	include, includeErrs := resolvePatterns(includePatterns)
	errs = append(errs, includeErrs...)

	var result []string
	for _, i := range include {
		if _, ok := pathSet[i]; !ok && !filter(i) {
			pathSet[i] = true
			result = append(result, i)
		}
	}
	return result, errs
}

func newPathFilter(skip []string) func(string) bool {
	filter := map[string]bool{}
	for _, name := range skip {
		filter[name] = true
	}

	return func(path string) bool {
		base := filepath.Base(path)
		if filter[base] || filter[path] {
			return true
		}
		return base != "." && base != ".." && strings.ContainsAny(base[0:1], "_.")
	}
}
