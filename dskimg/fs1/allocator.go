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

import "github.com/maxymania/anyfs/dskimg/ods"
import "github.com/maxymania/anyfs/dskimg/bitmap"
import "github.com/maxymania/anyfs/debug"
import "errors"

var badalloc = errors.New("No Allocation possible")


type AllocRange struct{
	Begin,End uint64
}

type fs_job struct{
	from,to,clear,free AllocRange
}
func (f *FileSystem) FreeMFTE(mfte *ods.MFTE) error {
	blank := new(ods.MFTE)
	e := f.MMFT.PutEntryLL(mfte.File_MFT,mfte.File_IDX,blank)
	if e!=nil { return e }
	_,e = f.FreeRangeSync(mfte.Begin_BLK,mfte.End_BLK)
	return e
}
func (f *FileSystem) ClearMFTE(mfte *ods.MFTE) error {
	beg,end := mfte.Begin_BLK,mfte.End_BLK
	mfte.Begin_BLK = 0
	mfte.End_BLK = 0
	e := f.MMFT.PutEntry(mfte)
	if e!=nil { return e }
	_,e = f.FreeRangeSync(beg,end)
	return e
}
func (f *FileSystem) dojob(job* fs_job) {
	status := false
	buf := []byte{}
	i,n,j,m := job.from.Begin,job.from.End,job.to.Begin,job.to.End
	for i<n && j<m {
		if !status { buf = make([]byte,f.SB.BlockSize); status = true }
		f.Device.ReadAt(buf,f.SB.Offset(i))
		f.Device.WriteAt(buf,f.SB.Offset(j))
		i++
		j++
	}
	status = false
	i,n = job.clear.Begin,job.clear.End
	for ; i<n; i++ {
		if !status { buf = make([]byte,f.SB.BlockSize); status = true }
		f.Device.WriteAt(buf,f.SB.Offset(i))
	}
	i,n = job.free.Begin,job.free.End
	if i<n {
		f.BMLck.Lock()
		defer f.BMLck.Unlock()
		f.FreeRange(i,n)
		//f.BitMap.Apply(buf,i,n,bitmap.FreeRange,true)
	}
}
func (f *FileSystem) GrowMFTE(mfte *ods.MFTE, nblocks uint64) (error,bool) {
	j := new(fs_job)
	debug.Println("GrowMFTE(",nblocks,") {")
	defer debug.Println("}GrowMFTE")
	debug.Println(" f.growMFTE...")
	e,more := f.growMFTE(mfte,nblocks,j)
	debug.Println(" f.growMFTE(",nblocks,j,") -> ",e,more)
	f.dojob(j)
	dirty := false
	if e!=nil { return e,dirty }
	for more {
		debug.Println(" f.addMFTE...")
		m2,nd,e := f.addMFTE(mfte)
		debug.Println(" f.addMFTE() -> ",nd,e)
		dirty = nd || dirty
		if e!=nil { return e,dirty }
		nblocks -= (mfte.End_BLK-mfte.Begin_BLK)
		mfte = m2
		*j = fs_job{}
		debug.Println(" f.growMFTE...")
		e,more = f.growMFTE(mfte,nblocks,j)
		debug.Println(" f.growMFTE(",nblocks,j,") -> ",e,more)
		f.dojob(j)
		if e!=nil { return e,dirty }
	}
	return nil,dirty
}

func (f *FileSystem) addMFTE(mfte *ods.MFTE) (*ods.MFTE,bool,error) {
	f.MFTLck.Lock()
	defer f.MFTLck.Unlock()
	m2,e := f.MMFT.Allocate(mfte.File_MFT)
	if e!=nil { return nil,false,e }
	m2.First_IDX  = mfte.File_IDX
	mfte.Next_IDX = m2.File_IDX
	e = f.MMFT.PutEntry(m2)
	if e!=nil { return nil,false,e }
	e  = f.MMFT.PutEntry(mfte)
	if e!=nil { return nil,true,e }
	return m2,true,nil
}
func (f *FileSystem) growMFTE(mfte *ods.MFTE, nblocks uint64, j* fs_job) (error,bool) {
	f.BMLck.Lock()
	defer f.BMLck.Unlock()
	if mfte.Begin_BLK==0 || mfte.Begin_BLK>=mfte.End_BLK {
		debug.Println("  f.AllocateRange...")
		ar,e := f.AllocateRange(nblocks)
		debug.Println("  f.AllocateRange(",nblocks,") -> ",ar,e)
		// TODO: Handle badalloc
		if e!=nil { return e,false }
		mfte.Begin_BLK = ar.Begin
		mfte.End_BLK   = ar.End
		j.clear = *ar
		e = f.MMFT.PutEntry(mfte)
		return e,false
	} else {
		diff := mfte.End_BLK-mfte.Begin_BLK
		if diff>=nblocks { return nil,false } /* Don't grow if not needed. */
		ddiff := (nblocks-diff)
		debug.Println("  f.AllocAppend...")
		ne,e := f.AllocAppend(mfte.End_BLK,ddiff)
		debug.Println("  f.AllocAppend(",mfte.End_BLK,ddiff,") -> ",ne,e)
		if e!=nil { return e,false }
		//fmt.Println("Append: ",&AllocRange{mfte.Begin_BLK,ne})
		ndiff := (ne-mfte.Begin_BLK)
		if ndiff>=nblocks {
			j.clear.Begin = mfte.End_BLK
			j.clear.End   = ne
			mfte.End_BLK  = ne
			e = f.MMFT.PutEntry(mfte)
			return e,false
		}
		
		dndiff := (nblocks-ndiff)
		minimum := ndiff+(dndiff/2)
		debug.Println("  f.AllocateBiggest...")
		ar,e := f.AllocateBiggest(nblocks,minimum)
		debug.Println("  f.AllocateBiggest(",nblocks,minimum,") -> ",ar,e)
		//fmt.Println("Biggest: ",ar)
		if e==badalloc {
			j.clear.Begin = mfte.End_BLK
			j.clear.End   = ne
			mfte.End_BLK  = ne
			e = f.MMFT.PutEntry(mfte)
			return e,ndiff<nblocks
		}
		if e!=nil { return e,false }
		j.from.Begin   = mfte.Begin_BLK
		j.from.End     = mfte.End_BLK
		j.to.Begin     = ar.Begin
		j.to.End       = ar.Begin+diff
		j.clear.Begin  = ar.Begin+diff
		j.clear.End    = ar.End
		j.free.Begin   = mfte.Begin_BLK
		j.free.End     = ne
		
		mfte.Begin_BLK = ar.Begin
		mfte.End_BLK   = ar.End
		e = f.MMFT.PutEntry(mfte)
		
		ndiff = ar.End-ar.Begin
		return nil,ndiff<nblocks
	}
	panic("unreachable")
}
func (f *FileSystem) AllocAppend(pos, n uint64) (uint64,error) {
	bl := (n+7)>>3
	bl += 2
	if bl>(1<<20) { bl = 1<<20 }
	buf := make([]byte,int(bl))
	end := f.SB.Block_Len
	posn := pos+n
	if posn>end { posn = end }
	if pos>posn { return 0,badalloc }
	if pos==posn { return pos,nil }
	
	return f.BitMap.Apply(buf,pos,posn,bitmap.AllocRange,true)
}
func (f *FileSystem) FreeRangeSync(pos, end uint64) (uint64,error) {
	f.BMLck.Lock()
	defer f.BMLck.Unlock()
	return f.FreeRange(pos,end)
}
func (f *FileSystem) FreeRange(pos, end uint64) (uint64,error) {
	n := end-pos
	bl := (n+7)>>3
	bl += 2
	if bl>(1<<20) { bl = 1<<20 }
	buf := make([]byte,int(bl))
	
	for {
		np,e := f.BitMap.Apply(buf,pos,end,bitmap.FreeRange,true)
		if e!=nil { return np,e }
		if np>=end { break }
		if np<=pos { break }
		pos = np
	}
	return pos,nil
}
func (f *FileSystem) AllocateRange(n uint64) (*AllocRange,error) {
	bl := (n+7)>>3
	bl += 2
	if bl>(1<<20) { bl = 1<<20 }
	buf := make([]byte,int(bl))
	
	pos := uint64(0)
	end := f.SB.Block_Len
	// min64(end,pos+n)
	
	for {
		debug.Println("SkipAllocated(",pos,"...",end,") ...")
		for {
			lp,e := f.BitMap.Apply(buf,pos,end,bitmap.ScanSetRange,false)
			if e!=nil { return nil,e }
			if lp==pos { break }
			pos = lp
		}
		debug.Println("SkipAllocated(...) -> ",pos,"...",end)
		goal := pos+n
		lp,e := f.BitMap.Apply(buf,pos,end,bitmap.ScanRange,false)
		if goal<=lp {
			_,e = f.BitMap.Apply(buf,pos,goal,bitmap.SetRange,true)
			if e!=nil { return nil,e }
			return &AllocRange{pos,goal},nil
		}
		if end<=lp { break }
		if pos==lp { break }
		pos = lp
	}
	return nil,badalloc
}
func (f *FileSystem) AllocateBiggest(n, minimum uint64) (*AllocRange,error) {
	bl := (n+7)>>3
	bl += 2
	if bl>(1<<20) { bl = 1<<20 }
	buf := make([]byte,int(bl))
	
	aro := new(AllocRange)
	arl := uint64(0)
	
	pos := uint64(0)
	end := f.SB.Block_Len
	// min64(end,pos+n)
	
	for {
		lp,e := f.BitMap.Apply(buf,pos,end,bitmap.ScanSetRange,false)
		if e!=nil { return nil,e }
		pos = lp
		goal := pos+n
		lp,e = f.BitMap.Apply(buf,pos,end,bitmap.ScanRange,false)
		if goal<=lp {
			_,e = f.BitMap.Apply(buf,pos,goal,bitmap.SetRange,true)
			*aro = AllocRange{pos,goal}
			return aro,e
		}
		al := lp-pos
		if al>arl {
			arl = al
			*aro = AllocRange{pos,lp}
		}
		if end<=lp { break }
		pos = lp
	}
	if arl<minimum { return nil,badalloc }
	_,err := f.BitMap.Apply(buf,aro.Begin,aro.End,bitmap.SetRange,true)
	
	return aro,err
}


