package fsproxy

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	os_path "path"
	"sync"
	"syscall"
)

type ListFunc func(p *Proxy, path string) ([]fuse.Dirent, fuse.Error)
type ReadAllFunc func(p *Proxy, path string) ([]byte, fuse.Error)
type LookupFunc func(p *Proxy, path string, intr fs.Intr) (fs.Node, fuse.Error)

type Proxy struct {
	Mountpoint string
	Source     string
	List       ListFunc
	Lookup     LookupFunc
	ReadAll    ReadAllFunc
	nextInode  uint64
	inodes     map[string]uint64
	lck        sync.Mutex
}

func New(mountpoint, source string) *Proxy {
	proxy := &Proxy{
		Mountpoint: mountpoint,
		Source:     source,
		List:       DefaultList,
		ReadAll:    DefaultReadAll,
		Lookup:     DefaultLookup,
		inodes:     map[string]uint64{},
		nextInode:  10,
	}

	return proxy
}

func (proxy *Proxy) Serve() error {
	err := fuse.Unmount(proxy.Mountpoint)
	if err != nil {
		log.Print(err)
	}

	connection, err := fuse.Mount(proxy.Mountpoint)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%+v", connection)

	result := fs.Serve(connection, proxy)

	log.Printf("Result %+v", result)

	return result
}

var DefaultList ListFunc = func(proxy *Proxy, path string) ([]fuse.Dirent, fuse.Error) {
	file, err := os.Open(path)
	if file == nil {
		fmt.Fprintf(os.Stderr, "cannot access %s: %v\n", path, err)
		os.Exit(1)
	}

	files, err := file.Readdirnames(-1)
	if err != nil {
		log.Fatalf("Could not read dir names %v", path)
	}

	dirs := []fuse.Dirent{}

	for j := 0; j < len(files); j++ {
		filepath := os_path.Join(path, files[j])
		info, err := os.Stat(filepath)
		if err != nil {
			return nil, fuse.Errno(syscall.ENOSYS)
		}
		var t fuse.DirentType

		//TODO: smarter type :D
		if info.IsDir() {
			t = fuse.DT_Dir
		}

		dirs = append(dirs, fuse.Dirent{
			Inode: proxy.inodeForPath(filepath),
			Name:  files[j],
			Type:  t,
		})

	}

	return dirs, nil
}

var DefaultReadAll ReadAllFunc = func(proxy *Proxy, path string) ([]byte, fuse.Error) {

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fuse.Errno(syscall.ENOSYS)
	}
	return bytes, nil
}

func (proxy *Proxy) Root() (fs.Node, fuse.Error) {
	return proxy.Directory(proxy.Source)
}

func (proxy *Proxy) Directory(path string) (fs.Node, fuse.Error) {
	return &Directory{Proxy: proxy, Path: path, Attributes: dirAttrsForPath(proxy, path)}, nil
}

func dirAttrsForPath(proxy *Proxy, path string) fuse.Attr {
	return fuse.Attr{
		Inode: proxy.inodeForPath(path),
		Mode:  os.ModeDir,
	}
}

func (proxy *Proxy) inodeForPath(path string) uint64 {
	proxy.lck.Lock()
	defer proxy.lck.Unlock()
	result, ok := proxy.inodes[path]
	if !ok {
		proxy.nextInode++
		result = proxy.nextInode
		proxy.inodes[path] = result
	}
	return result
}

var DefaultLookup LookupFunc = func(p *Proxy, path string, intr fs.Intr) (fs.Node, fuse.Error) {
	fmt.Println("In default lookup for ", path)

	info, err := os.Stat(path)
	if err != nil {
		return nil, fuse.Errno(syscall.ENOSYS)
	}
	//TODO: smarter type :D
	if info.IsDir() {
		return &Directory{
			Proxy:      p,
			Path:       path,
			Attributes: dirAttrsForPath(p, path),
		}, nil
	} else {
		return &File{
			Proxy: p,
			Path:  path,
			Attributes: fuse.Attr{
				Inode: p.inodeForPath(path),
				Mode:  0555,
			},
		}, nil
	}

	return nil, fuse.Errno(syscall.ENOSYS)
}

type Directory struct {
	Proxy      *Proxy
	Path       string
	Name       string
	Attributes fuse.Attr
}

func (dir *Directory) Attr() fuse.Attr {
	return dir.Attributes
}

func (dir *Directory) ReadDir(fs.Intr) ([]fuse.Dirent, fuse.Error) {
	log.Printf("Reading directory for %+v", dir)
	return dir.Proxy.List(dir.Proxy, dir.Path)
}

func (dir *Directory) Lookup(path string, intr fs.Intr) (fs.Node, fuse.Error) {
	log.Printf("Looking up entry for %+v in %+v", path, dir)
	return dir.Proxy.Lookup(dir.Proxy, os_path.Join(dir.Path, path), intr)
}

type File struct {
	Proxy      *Proxy
	Path       string
	Attributes fuse.Attr
}

func (file *File) Attr() fuse.Attr {
	return file.Attributes
}

func (file *File) ReadAll(intr fs.Intr) ([]byte, fuse.Error) {
	bytes, err := ioutil.ReadFile(file.Path)
	if err != nil {
		return nil, fuse.Errno(syscall.ENOSYS)
	}
	return bytes, nil
}
