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

var invalidfiles = errors.New("Bas MFT Entry Allocation")

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
	
	z[0].PullHead(ppart)
	z[len(z)-1].PullTail(epart)
	return z,nil
}


