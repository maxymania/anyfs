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

package fs1

import "os"
import "io"
import "github.com/maxymania/anyfs/dskimg/ods"
import "errors"

var invalidfiles = errors.New("Bad MFT Entry Allocation")

var EIO = errors.New("IO_ERROR")

type FileRange struct{
	Device *os.File
	Pos    int64
	Len    int64
}
func (f* FileRange) FileRange() *FileRange{ return f }
func (f* FileRange) PullHead(i int64) *FileRange {
	f.Pos+=i
	f.Len-=i
	return f
}
func (f* FileRange) PullTail(i int64) *FileRange {
	f.Len-=i
	return f
}
func (f* FileRange) ReadObj(p []byte) (n int,err error) {
	if f.Len<int64(len(p)) { p = p[:f.Len] }
	n,_ = f.Device.ReadAt(p,f.Pos)
	if n<len(p) { err = EIO }
	return
}
func (f* FileRange) WriteObj(p []byte) (n int,err error) {
	if f.Len<int64(len(p)) { p = p[:f.Len] }
	n,_ = f.Device.WriteAt(p,f.Pos)
	if n<len(p) { err = EIO }
	return
}


func ReadFileRanges(rr []*FileRange,p []byte) (n int,err error) {
	n = 0
	for _,r := range rr {
		rn,e := r.ReadObj(p)
		n+=rn
		p = p[:rn]
		if e!=nil { err = e; return }
	}
	return n,nil
}
func WriteFileRanges(rr []*FileRange,p []byte) (n int,err error) {
	n = 0
	for _,r := range rr {
		rn,e := r.WriteObj(p)
		n+=rn
		p = p[:rn]
		if e!=nil { err = e; return }
	}
	return n,nil
}


type FileBlockRange struct {
	Device *os.File
	Block  uint32
	Begin  uint64
	End    uint64
}
func (fbr* FileBlockRange) FileRange() *FileRange {
	return &FileRange{
		fbr.Device,
		int64(fbr.Begin*uint64(fbr.Block)),
		int64((fbr.End-fbr.Begin)*uint64(fbr.Block)),
	}
}

type File struct{
	FS   *FileSystem
	MFT  uint32
	FID  uint32
}
func (f *File) GetMFTE() (*ods.MFTE,error) {
	mfte,e := f.FS.MMFT.GetEntry(f.MFT,f.FID)
	if mfte.First_IDX!=f.FID { return nil,invalidfiles } /* Invalid file-head. */
	return mfte,e
}
func (f *File) offset(begin, end, voff uint64, mfte *ods.MFTE, rp* FileBlockRange) uint64{
	bb := mfte.Begin_BLK
	eb := mfte.End_BLK
	if eb<bb { eb=bb } /* just in case... */
	tbeg := bb + (begin-voff)
	tend := bb + (end-voff)
	if eb < tend { tend = eb }
	rp.Begin = tbeg
	rp.End   = tend
	return voff + (tend-bb)
}
func (f *File) Foffset(bblk,eblk uint64) ([]FileBlockRange,error){
	gec,e := f.FS.MMFT.GetEntryChain(f.MFT,f.FID)
	if e!=nil { return nil,e }
	bidx,fi := gec.FindBlockOffset(bblk)
	if fi<0 { return nil,io.EOF }
	
	ran := make([]FileBlockRange,0,4)
	rp := new(FileBlockRange)
	rp.Device = f.FS.Device
	rp.Block  = f.FS.SB.BlockSize
	
	for bblk<eblk {
		mfte,e := f.FS.MMFT.GetEntry(f.MFT,bidx)
		if e!=nil { return nil,e }
		zblk := f.offset(bblk,eblk,gec.Off_BLK[fi],mfte,rp)
		if zblk<bblk { break }
		bblk = zblk
		ran = append(ran,*rp)
		fi++
		if fi>=len(gec.Off_BLK) { break } /* No more entries. */
		bidx = gec.Indeces[fi]
	}
	
	return ran,nil
}
func (f *File) Franges(pos int64, l int) ([]*FileRange,error){
	end := int64(l)+pos
	mfte,e := f.GetMFTE()
	if e!=nil { return nil,e }
	if end > mfte.FileSize { end = mfte.FileSize }
	return f.FrangesLL(pos,end)
}
func (f *File) FrangesLL(pos,end int64) ([]*FileRange,error){
	bz := uint64(f.FS.SB.BlockSize)
	sbz := int64(f.FS.SB.BlockSize)
	
	
	ppart := pos%sbz
	epart := (sbz-(pos%sbz))%sbz
	
	pblk := uint64(pos)/bz
	eblk := (uint64(end)+bz-1)/bz
	
	ff,e := f.Foffset(pblk,eblk)
	if e!=nil { return nil,e }
	
	z := make([]*FileRange,len(ff))
	for i,f := range ff { z[i]=f.FileRange() }
	
	if len(z)>0{
		z[0].PullHead(ppart)
		z[len(z)-1].PullTail(epart)
	}
	return z,nil
}

// Purge file segments, that are not longer needed. (truncate)
func (f *File) ShrinkDsk() error {
	f.FS.MFTLck.Lock()
	defer f.FS.MFTLck.Unlock()
	mfte,e := f.GetMFTE()
	if e!=nil { return e }
	
	bz := uint64(f.FS.SB.BlockSize)
	blks := (uint64(mfte.FileSize)+bz-1)/bz
	
	gec,e := f.FS.MMFT.GetEntryChain(f.MFT,f.FID)
	if e!=nil { return e }
	
	if gec.TotalBLK <= blks {
		return nil
	}
	i := len(gec.Indeces)-1
	
	for {
		lb := gec.Off_BLK[i]
		bmfte,e := f.FS.MMFT.GetEntry(f.MFT,gec.Indeces[i])
		if e!=nil { return e }
		if lb >= blks {
			if i>0 {
				i--
				pmfte,e := f.FS.MMFT.GetEntry(f.MFT,gec.Indeces[i])
				if e!=nil { return e }
				pmfte.Next_IDX = 0
				e = f.FS.MMFT.PutEntry(pmfte)
				if e!=nil { return e }
				f.FS.FreeMFTE(bmfte)
				continue
			}else{
				f.FS.ClearMFTE(bmfte)
				break
			}
		}else{
			diff := bmfte.End_BLK-bmfte.Begin_BLK
			if bmfte.End_BLK<bmfte.Begin_BLK { break }
			cdif := blks-lb
			if cdif>=diff { break }
			oe := bmfte.End_BLK
			ne := bmfte.Begin_BLK+cdif
			bmfte.End_BLK = ne
			e := f.FS.MMFT.PutEntry(bmfte)
			if e!=nil { return e }
			f.FS.FreeRangeSync(ne,oe)
			break
		}
	}
	/* Flush Cache. */
	f.FS.MMFT.ResetEntryChain(f.MFT,f.FID)
	return nil
}
func (f *File) Grow(size int64) error {
	return f.sizectl(size,false,true)
}
func (f *File) Resize(size int64) error {
	return f.sizectl(size,true,true)
}
func (f *File) sizectl(size int64,shrink, grow bool) error {
	bz := uint64(f.FS.SB.BlockSize)
	blks := (uint64(size)+bz-1)/bz
	mfte,e := f.GetMFTE()
	if e!=nil { return e }
	if mfte.FileSize>size { /* Shrink */
		if !shrink { return nil }
		mfte.FileSize = size
		return f.FS.MMFT.PutEntry(mfte)
	}
	/* If the file size is not different, do nothing. */
	if mfte.FileSize==size { return nil }
	
	if !grow { return nil }
	
	gec,e := f.FS.MMFT.GetEntryChain(f.MFT,f.FID)
	if e!=nil { return e }
	if gec.TotalBLK < blks {
		i := len(gec.Indeces)-1
		needblk := blks-gec.Off_BLK[i]
		lmfte,e := f.FS.MMFT.GetEntry(f.MFT,gec.Indeces[i])
		if e!=nil { return e }
		e,_ = f.FS.GrowMFTE(lmfte,needblk)
		if e!=nil { return e }
		
		/* Flush Cache. */
		f.FS.MMFT.ResetEntryChain(f.MFT,f.FID)
	}
	
	/* Refresh entry, to make sure, we don't operate on a stale copy */
	mfte,e = f.GetMFTE()
	if e!=nil { return e }
	
	if mfte.FileSize==size { return nil } /* Nothing to do */
	
	mfte.FileSize = size
	return f.FS.MMFT.PutEntry(mfte)
}
func (f *File) Size() (int64,error) {
	mfte,e := f.GetMFTE()
	if e!=nil { return 0,e }
	return mfte.FileSize,nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	lp := len(p)
	r,e := f.Franges(off,lp)
	if e!=nil  { return 0,e }
	n,err = ReadFileRanges(r,p)
	if n<lp {
		if err==nil { err = io.EOF }
	} else {
		err = nil
	}
	return
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	lp := len(p)
	r,e := f.Franges(off,lp)
	if e!=nil  { return 0,e }
	n,err = WriteFileRanges(r,p)
	if n<lp {
		if err==nil { err = EIO }
	} else {
		err = nil
	}
	return
}

func (f *File) AsDirectory() (*ods.Directory,error) {
	return ods.NewDirectory(&AutoGrowingFile{f},int(f.FS.SB.DirSegSize))
}


type AutoGrowingFile struct{
	*File
}
func (f *AutoGrowingFile) WriteAt(p []byte, off int64) (n int, err error) {
	lp := len(p)
	f.Grow(off+int64(lp))
	r,e := f.Franges(off,lp)
	if e!=nil  { return 0,e }
	n,err = WriteFileRanges(r,p)
	if n<lp {
		if err==nil { err = EIO }
	} else {
		err = nil
	}
	return
}


