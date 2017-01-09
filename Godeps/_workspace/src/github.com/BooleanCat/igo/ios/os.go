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
}

//OSWrap is a wrapper around os that implements ios.OS
type OSWrap struct{}

//Rename is a wrapper around os.Rename()
func (osw *OSWrap) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

//Remove is a wrapper around os.Remove()
func (osw *OSWrap) Remove(path string) error {
	return os.Remove(path)
}

//Chmod is a wrapper around os.Chmod()
func (osw *OSWrap) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

//Chown is a wrapper around os.Chown()
func (osw *OSWrap) Chown(name string, uid, gid int) error {
	return os.Chown(name, uid, gid)
}

//OpenFile is a wrapper around os.OpenFile()
func (osw *OSWrap) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

//Stat is a wrapper around os.Stat()
func (osw *OSWrap) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

//FindProcess is a wrapper around os.FindProcess()
func (osw *OSWrap) FindProcess(pid int) (Process, error) {
	process, err := os.FindProcess(pid)
	return &ProcessWrap{process: process}, err
}
