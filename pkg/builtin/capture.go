package builtin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func insertAssetAtPath(filePath string, a func() (*asset, error)) error {
	if _, ok := _bindata[filePath]; ok {
		return fmt.Errorf("%q already exists", filePath)
	}

	dir := _bintree
	dirname, fname := path.Split(filePath)

	// Walk the directory path components, looking up the
	// corresponding entry in the bintree. By the end, we have the
	// bintree element we want to insert into.
	for _, p := range strings.Split(dirname, "/") {
		if p == "" || p == "." {
			continue
		}

		if dir.Func != nil {
			return fmt.Errorf("%q is not a directory", p)
		}

		entry := dir.Children[p]
		if entry == nil {
			entry = &bintree{
				Func:     nil,
				Children: map[string]*bintree{},
			}
			dir.Children[p] = entry
		}

		// If the entry already has a node, we can't traverse
		// it because it is a file (i.e. a leaf in the tree).
		if entry.Func != nil {
			return fmt.Errorf("%q is not a directory", p)
		}

		dir = dir.Children[p]
	}

	dir.Children[fname] = &bintree{
		Func:     a,
		Children: nil,
	}

	_bindata[filePath] = a

	return nil
}

type NameTransformFunc func(string) string

func WithPrefix(prefix string) func(string) string {
	return func(filePath string) string {
		return path.Join(prefix, filePath)
	}
}

func CaptureAsset(filePath string, xfrm ...NameTransformFunc) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil
	}

	for _, x := range xfrm {
		filePath = x(filePath)
	}

	loader := func() (*asset, error) {
		a := &asset{
			bytes: data,
			info: bindataFileInfo{
				name:    filePath,
				size:    info.Size(),
				mode:    info.Mode(),
				modTime: info.ModTime(),
			},
		}

		return a, nil
	}

	return insertAssetAtPath(filePath, loader)
}

func CaptureAssets(dirPath string, xfrm ...NameTransformFunc) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return CaptureAsset(dirPath, xfrm...)
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		return CaptureAsset(path, xfrm...)
	})
}
