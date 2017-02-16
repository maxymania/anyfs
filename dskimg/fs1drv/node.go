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

func opennode(fs *fs1.FileSystem, ent ods.DirectoryEntryValue) (bool,nodefs.Node,fuse.Status) {
	file := fs.GetFile(ent.File_MFT,ent.File_IDX)
	mfte,e := file.GetMFTE()
	if e!=nil { return false,nil,fuse.EIO }
	if mfte.Cookie!=ent.Cookie { return false,nil,fuse.EINVAL } /* XXX Cookie error: delete the entry. */
	switch mfte.FileType {
	case ods.FT_FILE:{
		n := &FileNode{nodefs.NewDefaultNode(),&fs1.AutoGrowingFile{file}}
		return false,n,fuse.OK
		}
	case ods.FT_DIR:{
		d,e := file.AsDirectory()
		if e!=nil { return false,nil,fuse.EIO }
		dn := new(DirNode)
		dn.Node = nodefs.NewDefaultNode()
		dn.Backing = file
		dn.Dir = d
		return true,dn,fuse.OK
		}
	case ods.FT_FIFO:{
		mm := new(ModeNode)
		mm.Node = nodefs.NewDefaultNode()
		mm.Attr.Mode = fuse.S_IFIFO | 0666
		//attr_now(&(mm.Attr))
		return true,mm,fuse.OK
		}
	}
	return false,nil,fuse.EIO
}


type ModeNode struct{
	nodefs.Node
	Attr fuse.Attr
}

func (m *ModeNode) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) (fuse.Status) {
	*out = m.Attr
	return fuse.OK
}

/* Mode-Nodes must be kept in memory in order to reserve their INODE number. */
func (m *ModeNode) Deletable() bool { return false }


