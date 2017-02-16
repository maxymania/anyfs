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

package ods

import "io"
import "encoding/binary"
import "errors"

import "github.com/hashicorp/golang-lru"
import "github.com/maxymania/anyfs/dskimg"


var ELongname = errors.New("Filename too long")

// DirectoryEntryValue are 17 byte long
type DirectoryEntryValue struct{
	File_MFT  uint32
	File_IDX  uint32
	Cookie    uint64
	FileType  uint8
}

// In encoded form it looks like this: 1 byte name_length; name; 17 byte File-Link
type DirectoryEntry struct{
	Name  string
	Value DirectoryEntryValue
}

func growDirectoryEntry_Array(de []DirectoryEntry) []DirectoryEntry{
	ca := cap(de)
	if len(de)==ca {
		if ca==0 {
			ca = 1
		} else if ca<128 {
			ca+=ca
		} else {
			ca+=128
		}
		nde := make([]DirectoryEntry,len(de),ca)
		copy(nde,de)
		return nde
	}
	return de
}
func resize_name_buf(nb []byte,size int) []byte {
	c := cap(nb)
	if c<size { return make([]byte,size) }
	return nb[:size]
}

func readDirEntries(src io.Reader) ([]DirectoryEntry,error) {
	buf := []byte{0}
	dea := []DirectoryEntry{}
	dev := new(DirectoryEntryValue)
	nb := []byte{}
	var err error
	for {
		_,e := src.Read(buf)
		if e!=nil { err = e; break }
		if buf[0]==0 { break }
		nb = resize_name_buf(nb,int(buf[0]))
		_,e = io.ReadFull(src,nb)
		if e!=nil { err = e; break }
		e = binary.Read(src,binary.BigEndian,dev)
		if e!=nil { err = e; break }
		dea = append(growDirectoryEntry_Array(dea),DirectoryEntry{string(nb),*dev})
	}
	if len(dea)>0 { err = nil }
	return dea,err
}
func writeDirEntries(ents []DirectoryEntry, dst io.Writer) error {
	buf := []byte{0}
	for _,ent := range ents {
		n := ent.Name
		nl := len(n)
		if nl>255 { return ELongname }
		if nl==0 { return ELongname }
		buf[0] = byte(nl)
		_,e := dst.Write(buf)
		if e!=nil { return e }
		_,e = dst.Write([]byte(n))
		if e!=nil { return e }
		e = binary.Write(dst,binary.BigEndian,ent.Value)
		if e!=nil { return e }
	}
	buf[0] = 0
	dst.Write(buf)
	return nil
}
func length_Dirents(ents []DirectoryEntry) int {
	i := 1 /* null-byte on the end. */
	for _,ent := range ents {
		i+= len(ent.Name) + 18 /* 1 byte name_length; name; 17 byte File-Link */
	}
	return i
}

type dir_cache_ent struct{
	idx int64
	dev DirectoryEntryValue
}

// This is a directory. Access must be synchronized.
type Directory struct{
	File   RAS
	Buf    *dskimg.FixedIO
	Segsz  int
	
	name_pos  *lru.Cache
	name_ent  *lru.Cache
}
func NewDirectory(file RAS,segsize int) (*Directory,error){
	d := new(Directory)
	d.File  = file
	d.Buf   = &dskimg.FixedIO{make([]byte,segsize),0}
	d.Segsz = segsize
	var e error
	d.name_pos,e = lru.New(256)
	if e!=nil { return nil,e }
	d.name_ent,e = lru.NewWithEvict(256,func(key interface{},value interface{}) {
		ce := value.(*dir_cache_ent)
		d.name_pos.Add(key,ce.idx) /* If cache one evicts, add it to two. */
	})
	if e!=nil { return nil,e }
	return d,nil
}

// Works like NewDirectory, but creates no cache.
// Only ReadDir(), WriteDir(), ListUp() and IsEmpty() might be used.
func NewDirectoryLite(file RAS,segsize int) *Directory{
	d := new(Directory)
	d.File  = file
	d.Buf   = &dskimg.FixedIO{make([]byte,segsize),0}
	d.Segsz = segsize
	return d
}

func (d *Directory) ReadDir(i int64) ([]DirectoryEntry,error) {
	e := d.Buf.ReadIndex(i,d.File)
	if e!=nil { return nil,e }
	return readDirEntries(d.Buf)
}
func (d *Directory) WriteDir(i int64,des []DirectoryEntry) error {
	d.Buf.Pos = 0
	e := writeDirEntries(des,d.Buf)
	if e!=nil { return e }
	return d.Buf.WriteIndex(i,d.File)
}
func (d *Directory) Search(name string) (fidx int64,dirent DirectoryEntryValue,err error) {
	i := int64(0)
	sp := false
	if entry,ok := d.name_ent.Get(name); ok {
		ce := entry.(*dir_cache_ent)
		dirent = ce.dev
		fidx   = ce.idx
		return
	}
	if idx,ok := d.name_ent.Get(name); ok {
		i  = idx.(int64)
		sp = true /* Only search one index. */
	}
	
	for {
		ents,e := d.ReadDir(i)
		if e!=nil { err = e; return }
		for _,ent := range ents {
			N := ent.Name
			d.name_ent.Add(N,&dir_cache_ent{i,ent.Value})
			if N==name { fidx = i; dirent = ent.Value; return }
		}
		if sp { err = io.EOF; return }
		i++
	}
}
func (d *Directory) Delete(name string) (dirent DirectoryEntryValue,err error) {
	var index int64
	index,dirent,err = d.Search(name)
	if err!=nil { return }
	ents,e := d.ReadDir(index)
	if e!=nil { err = e; return }
	err = io.EOF
	for ri,ent := range ents {
		if ent.Name!=name { continue }
		nlen := len(ents)-1
		if ri<nlen {
			copy(ents[ri:],ents[ri+1:])
		}
		ents = ents[:nlen]
		err = d.WriteDir(index,ents)
		d.name_ent.Remove(name)
		d.name_pos.Remove(name)
		break
	}
	return
}
func (d *Directory) Add(dir DirectoryEntry) error {
	if len(dir.Name)>255 || dir.Name=="" { return ELongname }
	if length_Dirents([]DirectoryEntry{dir})>d.Segsz { return ELongname } /* Just in case */
	for i:=int64(0); true; i++ {
		arr,e := d.ReadDir(i)
		arr = append(arr,dir)
		lng := length_Dirents(arr)
		if lng>d.Segsz { continue }
		e = d.WriteDir(i,arr)
		if e==nil {
			d.name_ent.Add(dir.Name,&dir_cache_ent{i,dir.Value})
		}
		return e
	}
	panic("unreachable")
}
func (d *Directory) ListUp(dest chan <- DirectoryEntry) {
	defer close(dest)
	for i:=int64(0); true; i++ {
		arr,e := d.ReadDir(i)
		if e!=nil { break }
		for _,o := range arr { dest <- o }
	}
}
func (d* Directory) IsEmpty() bool{
	for i:=int64(0); true; i++ {
		arr,e := d.ReadDir(i)
		if e!=nil { return true }
		if len(arr)>0 { return false }
	}
	panic("unreachable")
	
}


