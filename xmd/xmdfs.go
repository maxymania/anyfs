/*
 * MIT License
 * 
 * Copyright (c) 2017 Simon Schmidt
 * 
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 * 
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 * 
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
package xmd

import (
	"time"
	"path/filepath"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
)

type fileObj struct{
	nodefs.File
	path string
}
func (f* fileObj) InnerFile() nodefs.File { return f.File }
func (f* fileObj) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	readFile(f.path)
	return f.File.Read(dest,off)
}
func (f* fileObj) Write(data []byte, off int64) (written uint32, code fuse.Status) {
	writeFile(f.path)
	return f.File.Write(data,off)
}
func (f* fileObj) Truncate(size uint64) fuse.Status {
	writeFile(f.path)
	return f.File.Truncate(size)
}
func (f* fileObj) Chown(uid uint32, gid uint32) fuse.Status {
	updateMetadata(f.path)
	return f.File.Chown(uid,gid)
}
func (f* fileObj) Chmod(perms uint32) fuse.Status {
	updateMetadata(f.path)
	return f.File.Chmod(perms)
}
func (f* fileObj) Utimens(atime *time.Time, mtime *time.Time) fuse.Status {
	updateMetadata(f.path)
	return f.File.Utimens(atime,mtime)
}
func (f* fileObj) Release() {
	go closeFile(f.path)
	f.File.Release()
}

type wrappedFS struct{
	pathfs.FileSystem
	root string
}

func NewXmdLoopbackFileSystem(root string) pathfs.FileSystem {
	absr,e := filepath.Abs(root)
	if e!=nil { panic(e) }
	fs := pathfs.NewLoopbackFileSystem(absr)
	return &wrappedFS{fs,absr}
}

func (fs *wrappedFS) GetPath(relPath string) string {
	return filepath.Join(fs.root, relPath)
}

func (fs *wrappedFS) Chmod(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Chmod(name,mode,context)
	if code==fuse.OK {
		updateMetadata(fs.GetPath(name))
	}
	return
}

func (fs *wrappedFS) Chown(name string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Chown(name,uid,gid,context)
	if code==fuse.OK {
		updateMetadata(fs.GetPath(name))
	}
	return
}

func (fs *wrappedFS) Utimens(name string, Atime *time.Time, Mtime *time.Time, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Utimens(name,Atime,Mtime,context)
	if code==fuse.OK {
		updateMetadata(fs.GetPath(name))
	}
	return
}


func (fs *wrappedFS) Truncate(name string, size uint64, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Truncate(name,size,context)
	if code==fuse.OK {
		updateMetadata(fs.GetPath(name))
	}
	return
}

func (fs *wrappedFS) Access(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Access(name,mode,context)
	if code==fuse.OK {
		n := fs.GetPath(name)
		openFile(n)
		closeFile(n)
	}
	return
}

func (fs *wrappedFS) Link(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Link(oldName,newName,context)
	if code==fuse.OK {
		n := fs.GetPath(newName)
		linkFile(n)
	}
	return
}

func (fs *wrappedFS) Mkdir(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Mkdir(name,mode,context)
	if code==fuse.OK {
		createFile(fs.GetPath(name))
	}
	return
}

func (fs *wrappedFS) Mknod(name string, mode uint32, dev uint32, context *fuse.Context) (code fuse.Status) {
	code = fs.FileSystem.Mknod(name,mode,dev,context)
	if code==fuse.OK {
		createFile(fs.GetPath(name))
	}
	return
}

func (fs *wrappedFS) Rename(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	fakeDeleteFile(fs.GetPath(newName))
	code = fs.FileSystem.Rename(oldName,newName,context)
	if code==fuse.OK {
		renameFile(fs.GetPath(newName))
	}
	return
}

func (fs *wrappedFS) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	deleteFile(fs.GetPath(name))
	code = fs.FileSystem.Rmdir(name,context)
	return
}

func (fs *wrappedFS) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	deleteFile(fs.GetPath(name))
	code = fs.FileSystem.Unlink(name,context)
	return
}

func (fs *wrappedFS) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	file, code = fs.FileSystem.Open(name, flags, context)
	if code==fuse.OK {
		n := fs.GetPath(name)
		openFile(n)
		file = &fileObj{file,n}
	}
	return
}

func (fs *wrappedFS) GetXAttr(name string, attribute string, context *fuse.Context) (data []byte, code fuse.Status) {
	switch attribute {
	case "user.info.json":
		return extract(name),fuse.OK
	}
	return fs.FileSystem.GetXAttr(name,attribute,context)
}

func (fs *wrappedFS) ListXAttr(name string, context *fuse.Context) (attributes []string, code fuse.Status) {
	return []string{
		"user.info.json",
	},fuse.OK
}

//----------------------------
