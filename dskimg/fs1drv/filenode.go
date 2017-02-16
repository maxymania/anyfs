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
import "os"

import "fmt"

const ANYWRITE = uint32(os.O_WRONLY | os.O_RDWR | os.O_APPEND)

func read(f* fs1.AutoGrowingFile,dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	if len(dest)==0 { return fuse.ReadResultData([]byte{}),fuse.OK }
	l,_ := f.Franges(off,len(dest))
	if len(l)==0 { return nil,fuse.EIO }
	if len(l)==1 {
		ll := l[0]
		return fuse.ReadResultFd(ll.Device.Fd(),ll.Pos,int(ll.Len)),fuse.OK
	}
	n,e := fs1.ReadFileRanges(l,dest)
	if e!=nil { return nil,fuse.ToStatus(e) }
	return fuse.ReadResultData(dest[:n]),fuse.OK
}
func write(f* fs1.AutoGrowingFile,data []byte, off int64) (uint32, fuse.Status) {
	if len(data)==0 { return 0,fuse.OK }
	n,_ := f.WriteAt(data,off)
	if n==0 { return 0,fuse.EIO }
	return uint32(n),fuse.OK
}

type FileNode struct{
	nodefs.Node
	Backing *fs1.AutoGrowingFile
}
func (f *FileNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) (code fuse.Status) {
	if file!=nil { return f.Node.GetAttr(out, file, context) }
	mfte,e := f.Backing.GetMFTE()
	if e!=nil { return fuse.EIO }
	out.Mode = fuse.S_IFREG | 0666
	out.Size = uint64(mfte.FileSize)
	return fuse.OK
}
func (f *FileNode) Open(flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	if (flags&uint32(os.O_TRUNC))!=0 {
		f.Backing.Resize(0)
	}
	var fobj nodefs.File = &FileFile{nodefs.NewDefaultFile(),f.Backing}
	if (flags&ANYWRITE)==0 {
		fobj = nodefs.NewReadOnlyFile(fobj)
		//fobj = nodefs.NewDataFile([]byte("Test1\n"))
	}
	return fobj,fuse.OK
}
func (f *FileNode) Access(mode uint32, context *fuse.Context) (fuse.Status) {
	return fuse.OK
}


func (f *FileNode) Read(file nodefs.File, dest []byte, off int64, context *fuse.Context) (fuse.ReadResult, fuse.Status) {
	if file!=nil { return f.Node.Read(file,dest,off,context) }
	return read(f.Backing,dest,off)
}
func (f *FileNode) Write(file nodefs.File, data []byte, off int64, context *fuse.Context) (uint32, fuse.Status) {
	if file!=nil { return f.Node.Write(file,data,off,context) }
	return write(f.Backing,data,off)
}
func (f *FileNode) Truncate(file nodefs.File, size uint64, context *fuse.Context) (fuse.Status) {
	if file!=nil { return f.Node.Truncate(file,size,context) }
	e := f.Backing.Resize(int64(size))
	if e!=nil { return fuse.EIO }
	return fuse.OK
}

type FileFile struct{
	nodefs.File
	Backing *fs1.AutoGrowingFile
}
func (f* FileFile) String() string {
	return fmt.Sprint("FileObject(",f.Backing.MFT,",",f.Backing.FID,")")
}
func (f* FileFile) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	return read(f.Backing,dest,off)
}
func (f* FileFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	return write(f.Backing,data,off)
}
func (f* FileFile) Truncate(size uint64) fuse.Status {
	e := f.Backing.Resize(int64(size))
	if e!=nil { return fuse.EIO }
	return fuse.OK
}
func (f* FileFile) GetAttr(out *fuse.Attr) fuse.Status {
	mfte,e := f.Backing.GetMFTE()
	if e!=nil { return fuse.EIO }
	out.Mode = fuse.S_IFREG | 0666
	out.Size = uint64(mfte.FileSize)
	return fuse.OK
}
func (f *FileFile) Flush() fuse.Status { return fuse.OK }
func (f *FileFile) Fsync(flags int) fuse.Status { return fuse.OK }


