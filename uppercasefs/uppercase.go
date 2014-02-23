package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/aaronblohowiak/fsproxy"
	"io/ioutil"
	"log"
	"strings"
	"syscall"
)

type UpperCaseFile struct {
	fsproxy.File
}

func (file *UpperCaseFile) ReadAll(intr fs.Intr) ([]byte, fuse.Error) {
	bytes, err := ioutil.ReadFile(file.Path)
	if err != nil {
		return nil, fuse.Errno(syscall.ENOSYS)
	}

	return []byte(strings.ToUpper(string(bytes))), nil
}

func main() {
	//TODO: use flag / args instead of hardcoding these values

	proxy := fsproxy.New(
		"/Users/aaronblohowiak/projects/golang/test3",
		"/Users/aaronblohowiak/projects/golang/src")

	proxy.Lookup = func(p *fsproxy.Proxy, path string, intr fs.Intr) (fs.Node, fuse.Error) {
		node, err := fsproxy.DefaultLookup(p, path, intr)
		if err != nil {
			return nil, err
		}

		if file, ok := node.(*fsproxy.File); ok {
			return &UpperCaseFile{*file}, nil
		}

		return node, err
	}

	err := proxy.Serve()
	if err != nil {
		log.Fatal(err)
	}
}
