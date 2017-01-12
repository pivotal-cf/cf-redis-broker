package ios

import "os"

//OS is an interface around os
type OS interface {
	Rename(string, string) error
	Remove(string) error
	Chmod(string, os.FileMode) error
	Chown(string, int, int) error
	OpenFile(string, int, os.FileMode) (*os.File, error)
	Stat(string) (os.FileInfo, error)
	FindProcess(int) (Process, error)
	Getwd() (string, error)
}

//Real is a wrapper around os that implements ios.OS
type Real struct{}

//New creates a struct that behaves like the os package
func New() *Real {
	return new(Real)
}

//Rename is a wrapper around os.Rename()
func (*Real) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

//Remove is a wrapper around os.Remove()
func (*Real) Remove(path string) error {
	return os.Remove(path)
}

//Chmod is a wrapper around os.Chmod()
func (*Real) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

//Chown is a wrapper around os.Chown()
func (*Real) Chown(name string, uid, gid int) error {
	return os.Chown(name, uid, gid)
}

//OpenFile is a wrapper around os.OpenFile()
func (*Real) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

//Stat is a wrapper around os.Stat()
func (*Real) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

//FindProcess is a wrapper around os.FindProcess()
func (*Real) FindProcess(pid int) (Process, error) {
	process, err := os.FindProcess(pid)
	return &ProcessReal{process: process}, err
}

//Getwd is a wrapper around os.Getwd()
func (*Real) Getwd() (string, error) {
	return os.Getwd()
}
