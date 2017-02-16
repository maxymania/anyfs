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

package fs1drv

import "github.com/hanwen/go-fuse/fuse"
import "github.com/hanwen/go-fuse/fuse/nodefs"

import "github.com/maxymania/anyfs/dskimg/fs1"
import "github.com/maxymania/anyfs/dskimg/ods"

import "syscall"

import "sync"
import "io"

func extendArray(arr []fuse.DirEntry) []fuse.DirEntry{
	ca := cap(arr)
	le := len(arr)
	if ca==le {
		if ca<16 {
			ca = 16
		}else if ca<128 {
			ca+=ca
		}else{
			ca+=128
		}
		narr := make([]fuse.DirEntry,le,ca)
		copy(narr,arr)
		return narr
	}
	return arr
}

type DirNode struct{
	nodefs.Node
	Backing *fs1.File
	Dir     *ods.Directory
	Lock    sync.Mutex
}
func (d *DirNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) (code fuse.Status) {
	out.Mode = fuse.S_IFDIR | 0777
	return fuse.OK
}
func (d *DirNode) Lookup(out *fuse.Attr, name string, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	ino := d.Inode()
	c := ino.GetChild(name)
	if c!=nil {
		code := c.Node().GetAttr(out,nil,context)
		if !code.Ok() { c = nil }
		return c,code
	}
	_,ent,e := d.Dir.Search(name)
	if e==io.EOF { return nil,fuse.ENOENT }
	if e!=nil { return nil,fuse.ENOENT }
	
	dir,nd,st := opennode(d.Backing.FS,ent)
	if !st.Ok() { return nil,st }
	st = nd.GetAttr(out,nil,context)
	
	cld := ino.NewChild(name,dir,nd)
	
	if !st.Ok() { cld = nil }
	return cld,st
}
func (d *DirNode) OpenDir(context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	arr := []fuse.DirEntry{}
	
	cho := make(chan ods.DirectoryEntry,10)
	go d.Dir.ListUp(cho)
	
	for rdi := range cho {
		var ent fuse.DirEntry
		ent.Name = rdi.Name
		switch rdi.Value.FileType {
		case ods.FT_FILE:
			ent.Mode = fuse.S_IFREG
		case ods.FT_DIR:
			ent.Mode = fuse.S_IFDIR
		case ods.FT_FIFO:
			ent.Mode = fuse.S_IFIFO
		default: continue
		}
		arr = append(extendArray(arr),ent)
	}
	return arr,fuse.OK
}
func (d *DirNode) mkobj(name string, ft uint8) (ent ods.DirectoryEntryValue, code fuse.Status) {
	d.Lock.Lock()
	defer d.Lock.Unlock()
	_,_,e := d.Dir.Search(name)
	if e!= io.EOF { code = fuse.Status(syscall.EEXIST); return }
	f,e := d.Backing.FS.CreateFile(ft)
	if e!=nil { code = fuse.EIO; return }
	mfte,e := f.GetMFTE()
	if e!=nil { code = fuse.EIO; return }
	ent.File_MFT = mfte.File_MFT
	ent.File_IDX = mfte.File_IDX
	ent.Cookie   = mfte.Cookie
	ent.FileType = mfte.FileType
	e = d.Dir.Add(ods.DirectoryEntry{name,ent})
	if e!=nil { code = fuse.EIO; return }
	code = fuse.OK
	return
}
func (d *DirNode) Mkdir(name string, mode uint32, context *fuse.Context) (*nodefs.Inode,fuse.Status) {
	ino := d.Inode()
	ent,st := d.mkobj(name,ods.FT_DIR)
	if !st.Ok() { return nil,st }
	dir,nd,st := opennode(d.Backing.FS,ent)
	if !st.Ok() { return nil,st }
	return ino.NewChild(name,dir,nd),fuse.OK
}
func (d *DirNode) Mknod(name string, mode uint32, dev uint32, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	ino := d.Inode()
	ft := uint8(0)
	switch mode & syscall.S_IFMT {
	case fuse.S_IFREG:
		ft = ods.FT_FILE
	case fuse.S_IFIFO:
		ft = ods.FT_FIFO
	default:
		return nil,fuse.EINVAL
	}
	ent,st := d.mkobj(name,ft)
	if !st.Ok() { return nil,st }
	dir,nd,st := opennode(d.Backing.FS,ent)
	if !st.Ok() { return nil,st }
	return ino.NewChild(name,dir,nd),fuse.OK
}
func (d *DirNode) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File,*nodefs.Inode,fuse.Status) {
	ino := d.Inode()
	ent,st := d.mkobj(name,ods.FT_FILE)
	if !st.Ok() { return nil,nil,st }
	dir,nd,st := opennode(d.Backing.FS,ent)
	if !st.Ok() { return nil,nil,st }
	fobj,st := nd.Open(flags,context)
	if !st.Ok() {
		nd.OnForget()
		return nil,nil,st
	}
	return fobj,ino.NewChild(name,dir,nd),fuse.OK
}
func (d *DirNode) Unlink(name string, context *fuse.Context) (fuse.Status) {
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	_,ent,err := d.Dir.Search(name)
	if err!=nil {
		if ino.RmChild(name)!=nil { return fuse.OK } /* Transient Entries return OK. */
		return fuse.ENOENT
	}
	if ent.FileType==ods.FT_DIR { return fuse.Status(syscall.EISDIR) }
	
	_,err = d.Dir.Delete(name)
	if err==nil { return fuse.ENOENT }
	ino.RmChild(name)
	
	d.Backing.FS.Decrement(ent.File_MFT,ent.File_IDX)
	
	return fuse.OK
}
func (d *DirNode) Rmdir(name string, context *fuse.Context) (fuse.Status) {
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	_,ent,err := d.Dir.Search(name)
	if err!=nil { return fuse.ENOENT }
	if ent.FileType!=ods.FT_DIR { return fuse.ENOTDIR }
	ino.RmChild(name)
	{
		file := d.Backing.FS.GetFile(ent.File_MFT,ent.File_IDX)
		dir := file.AsDirectoryLite()
		if !dir.IsEmpty() { return fuse.Status(syscall.ENOTEMPTY) }
	}
	
	_,err = d.Dir.Delete(name)
	if err==nil { return fuse.ENOENT }
	
	d.Backing.FS.Decrement(ent.File_MFT,ent.File_IDX)
	
	return fuse.OK
}


