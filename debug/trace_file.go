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

package debug


import "github.com/hanwen/go-fuse/fuse"
import "github.com/hanwen/go-fuse/fuse/nodefs"
import "time"
import "fmt"

type TracerFile struct{
	nodefs.File
}
func (t *TracerFile) InnerFile() nodefs.File { return t.File }
func (t *TracerFile) Read(dest []byte, off int64) (res fuse.ReadResult, code fuse.Status) {
	res,code = t.File.Read(dest,off)
	fmt.Println(t.File,".Read([",len(dest),"],",off,") -> ",res.Size(),code)
	return
}

func (t *TracerFile) Write(data []byte, off int64) (written uint32, code fuse.Status) {
	written,code = t.File.Write(data,off)
	fmt.Println(t.File,".Write([",len(data),"],",off,") -> ",code)
	return
}

func (t *TracerFile) Flush() (code fuse.Status) {
	code = t.File.Flush()
	fmt.Println(t.File,".Flush() -> ",code)
	return
}

func (t *TracerFile) Release() {
	fmt.Println(t.File,".Release()")
	t.File.Release()
}
func (t *TracerFile) Fsync(flags int) (code fuse.Status) {
	code = t.File.Fsync(flags)
	fmt.Println(t.File,".Fsync(",flags,") -> ",code)
	return
}

func (t *TracerFile) Truncate(size uint64) (code fuse.Status) {
	code = t.File.Truncate(size)
	fmt.Println(t.File,".Truncate(",size,") -> ",code)
	return
}
func (t *TracerFile) GetAttr(out *fuse.Attr) (code fuse.Status) {
	code = t.File.GetAttr(out)
	fmt.Println(t.File,".GetAttr() -> ",out,code)
	return
}
func (t *TracerFile) Chown(uid uint32, gid uint32) (code fuse.Status) {
	code = t.File.Chown(uid,gid)
	fmt.Println(t.File,".Chown(",uid,",",gid,") -> ",code)
	return
}
func (t *TracerFile) Chmod(perms uint32) (code fuse.Status) {
	code = t.File.Chmod(perms)
	fmt.Println(t.File,".Chmod(",perms,") -> ",code)
	return
}
func (t *TracerFile) Utimens(atime *time.Time, mtime *time.Time) (code fuse.Status) {
	code = t.File.Utimens(atime,mtime)
	fmt.Println(t.File,".Utimens(",atime,",",mtime,") -> ",code)
	return
}
func (t *TracerFile) Allocate(off uint64, size uint64, mode uint32) (code fuse.Status) {
	code = t.File.Allocate(off,size,mode)
	fmt.Println(t.File,".Allocate(",off,",",size,",",mode,") -> ",code)
	return
}


