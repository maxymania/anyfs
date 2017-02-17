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

//import "github.com/maxymania/anyfs/debug"

import "github.com/maxymania/anyfs/dskimg/fs1"
import "github.com/maxymania/anyfs/dskimg/ods"
//import "time"

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
	uft := mode & syscall.S_IFMT
	switch uft {
	case fuse.S_IFREG:
		ft = ods.FT_FILE
	case fuse.S_IFIFO:
		ft = ods.FT_FIFO
	case syscall.S_IFSOCK:
		mm := new(ModeNode)
		mm.Node = nodefs.NewDefaultNode()
		mm.Attr.Mode = mode
		ci := ino.GetChild(name)
		if ci!=nil { return nil,fuse.Status(syscall.EEXIST) }
		return ino.NewChild(name,false,mm),fuse.OK
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
func (d *DirNode) Unlink(name string, context *fuse.Context) fuse.Status {
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
	if err!=nil { return fuse.ENOENT }
	ino.RmChild(name)
	
	mfte,e := d.Backing.FS.MMFT.GetEntry(ent.File_MFT,ent.File_IDX)
	if e==nil && mfte.Cookie==ent.Cookie {
		d.Backing.FS.Decrement(ent.File_MFT,ent.File_IDX)
	}
	
	return fuse.OK
}
func (d *DirNode) Rmdir(name string, context *fuse.Context) fuse.Status {
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
	if err!=nil { return fuse.ENOENT }
	
	mfte,e := d.Backing.FS.MMFT.GetEntry(ent.File_MFT,ent.File_IDX)
	if e==nil && mfte.Cookie==ent.Cookie {
		d.Backing.FS.Decrement(ent.File_MFT,ent.File_IDX)
	}
	
	return fuse.OK
}
func (d *DirNode) rename_in(oldName string, newName string, context *fuse.Context) fuse.Status {
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	{
		_,oent,oerr := d.Dir.Search(newName)
		if oerr!=nil && oent.FileType == ods.FT_DIR {
			return fuse.Status(syscall.EISDIR)
		}
	}
	_,ment,merr := d.Dir.Search(oldName)
	if merr==nil {
		oent,oerr := d.Dir.Delete(newName)
		ino.RmChild(newName)
		d.Dir.Add(ods.DirectoryEntry{newName,ment})
		on := ino.RmChild(oldName)
		if on!=nil { ino.AddChild(newName,on) }
		if oerr!=nil {
			mfte,e := d.Backing.FS.MMFT.GetEntry(oent.File_MFT,oent.File_IDX)
			if e==nil && mfte.Cookie==oent.Cookie {
				d.Backing.FS.Decrement(oent.File_MFT,oent.File_IDX)
			}
		}
		d.Dir.Delete(oldName)
		return fuse.OK
	}else{
		oin := ino.RmChild(oldName)
		if oin==nil { return fuse.ENOENT }
		ino.RmChild(newName)
		ino.AddChild(newName,oin)
		return fuse.OK
	}
	return fuse.ENOENT
}
func (d *DirNode) move_into(name string,ent ods.DirectoryEntryValue,nch *nodefs.Inode, context *fuse.Context) fuse.Status {
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	_,oent,oerr := d.Dir.Search(name)
	if oerr!=nil && oent.FileType == ods.FT_DIR {
		return fuse.Status(syscall.EISDIR)
	}
	oent,oerr = d.Dir.Delete(name)
	ino.RmChild(name)
	d.Dir.Add(ods.DirectoryEntry{name,ent})
	if nch!=nil { ino.AddChild(name,nch) }
	if oerr!=nil {
		mfte,e := d.Backing.FS.MMFT.GetEntry(oent.File_MFT,oent.File_IDX)
		if e==nil && mfte.Cookie==oent.Cookie {
			d.Backing.FS.Decrement(oent.File_MFT,oent.File_IDX)
		}
	}
	return fuse.OK
}
func (d *DirNode) move_out_1(name string) (nch *nodefs.Inode,ent ods.DirectoryEntryValue,err error) {
	pn := new(ModeNode)
	pn.Node = nodefs.NewDefaultNode()
	pn.Attr.Mode = fuse.S_IFDIR | 0777
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	
	_,ent,err = d.Dir.Search(name)
	if err!=nil { return }
	nch = ino.RmChild(name)
	ino.NewChild(name,true,pn)
	return
}
func (d *DirNode) move_out_2(name string) (err error) {
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	_,err = d.Dir.Delete(name)
	if err!=nil { ino.RmChild(name) }
	return
}
func (d *DirNode) move_out_rollback(name string,nch *nodefs.Inode) {
	ino := d.Inode()
	d.Lock.Lock()
	defer d.Lock.Unlock()
	ino.RmChild(name)
	ino.AddChild(name,nch)
}
func (d *DirNode) Rename(oldName string, newParent nodefs.Node, newName string, context *fuse.Context) fuse.Status {
	target,ok := newParent.(*DirNode)
	if !ok { return fuse.EINVAL }
	if d==target { return d.rename_in(oldName,newName,context) }
	ino,ent,err := d.move_out_1(oldName)
	if err!=nil {
		oldi := d.Inode().RmChild(oldName)
		if oldi==nil { return fuse.ENOENT }
		target.Inode().RmChild(newName)
		target.Inode().AddChild(newName,oldi)
		return fuse.OK
	}
	st := target.move_into(newName,ent,ino,context)
	if st.Ok() {
		d.move_out_2(oldName)
		return fuse.OK
	}else{
		d.move_out_rollback(oldName,ino)
		return st
	}
}
func (d *DirNode) link_ll(name string, ent ods.DirectoryEntryValue) (ok bool,err error){
	d.Lock.Lock()
	defer d.Lock.Unlock()
	_,_,oerr := d.Dir.Search(name)
	if oerr==io.EOF { return false,nil }
	err = d.Dir.Add(ods.DirectoryEntry{name,ent})
	ok = true
	return
}
func (d *DirNode) Link(name string, existing nodefs.Node, context *fuse.Context) (*nodefs.Inode, fuse.Status) {
	var ent ods.DirectoryEntryValue
	var mfte *ods.MFTE = nil
	var e error
	switch enode := existing.(type) {
	case *DirNode:
		if d.Backing.FS != enode.Backing.FS { return nil,fuse.EINVAL }
		mfte,e = enode.Backing.GetMFTE()
		if e!=nil { return nil,fuse.EIO }
	case *FileNode:
		if d.Backing.FS != enode.Backing.FS { return nil,fuse.EINVAL }
		mfte,e = enode.Backing.GetMFTE()
		if e!=nil { return nil,fuse.EIO }
	case *ReprNode:
		if d.Backing.FS != enode.Backing.FS { return nil,fuse.EINVAL }
		mfte,e = enode.Backing.GetMFTE()
		if e!=nil { return nil,fuse.EIO }
	case *ModeNode:
		nm := new(ModeNode)
		*nm = *enode
		nm.Node = nodefs.NewDefaultNode()
		oin := d.Inode().RmChild(name)
		if oin!=nil { d.Inode().AddChild(name,oin); return nil,fuse.Status(syscall.EEXIST) }
		return d.Inode().NewChild(name,false,nm),fuse.OK
	}
	if mfte!=nil {
		ent.File_MFT = mfte.File_MFT
		ent.File_IDX = mfte.File_IDX
		ent.Cookie   = mfte.Cookie
		ent.FileType = mfte.FileType
		ok,err := d.link_ll(name,ent)
		if err!=nil { return nil,fuse.EIO }
		if !ok { return nil,fuse.Status(syscall.EEXIST) }
		d.Backing.FS.Increment(mfte.File_MFT,mfte.File_IDX)
		dir,nd,st := opennode(d.Backing.FS,ent)
		if !st.Ok() { return nil,st }
		return d.Inode().NewChild(name,dir,nd),fuse.OK
	}
	return nil,fuse.EINVAL
}

