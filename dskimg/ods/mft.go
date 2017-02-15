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

import "sync"
import "encoding/binary"
import "github.com/maxymania/anyfs/dskimg"
import "errors"
import "github.com/hashicorp/golang-lru"
import "sort"
import "math/rand"

var allocmfte = errors.New("Bas MFT Entry Allocation")
var badmfte = errors.New("Bad MFT index")
var cormfte = errors.New("Corrupted MFT Entry")
var cecmfte = errors.New("Corrupted Chain of MFT Entries")

var nomfte = errors.New("No such MFT")

var mftenohead = errors.New("MFT Entry is not head chain")

const MFTE_SIZE = 64

const (
	FT_FILE = 0xf0+iota
	FT_DIR
	FT_FIFO
)

type MFTH struct{
	MFT_ID    uint32
	Num_BLK   uint32
	NextMFT   uint64
}

type MFTE struct{
	File_MFT  uint32
	File_IDX  uint32
	Cookie    uint64
	Begin_BLK uint64 /* First block of extend. */
	End_BLK   uint64 /* Last block. */
	Next_IDX  uint32 /* Next Element in Chain. */
	First_IDX uint32 /* First Element in Chain. */
	
	/*
	 * The following fields are only relevant for the head of a chain.
	 * On other segments of a chain these are to be ignored.
	 */
	FileSize  int64
	FileType  uint8
}

type MFTE_Chain struct{
	Indeces []uint32
	Off_BLK []uint64 /* Block offsets */
	TotalBLK  uint64 /* Total num of blocks */
}
func (chain *MFTE_Chain) FindBlockOffset(blkOff uint64) (uint32,int) {
	res := sort.Search(len(chain.Off_BLK),func(i int) bool{ return chain.Off_BLK[i] > blkOff })
	if res>0 {
		res--
		return chain.Indeces[res],res
	}
	return 0,-1
}

func growarray_u32(arr []uint32) []uint32{
	ln := len(arr)
	cp := cap(arr)
	if cp>ln { return arr }
	if cp<256 { cp<<=1 }else{ cp+=256 }
	narr := make([]uint32,ln,cp)
	copy(narr,arr)
	return narr
}
func growarray_u64(arr []uint64) []uint64{
	ln := len(arr)
	cp := cap(arr)
	if cp>ln { return arr }
	if cp<256 { cp<<=1 }else{ cp+=256 }
	narr := make([]uint64,ln,cp)
	copy(narr,arr)
	return narr
}

type MFT struct{
	Head  *MFTH
	Range RAS
	Buf   *dskimg.FixedIO
	Size  uint32
	EntriesPerBlock uint32
	
	list_cache  *lru.TwoQueueCache
	entry_cache *lru.TwoQueueCache
}
func NewMFT(r RAS,blockSize uint32) (*MFT,error) {
	mft := new(MFT)
	
	var e error
	mft.list_cache,e = lru.New2Q(256)
	if e!=nil { return nil,e }
	mft.entry_cache,e = lru.New2Q(256)
	if e!=nil { return nil,e }
	
	mft.Head = new(MFTH)
	mft.Range = r
	mft.Buf = &dskimg.FixedIO{make([]byte,MFTE_SIZE),0}
	mft.Size = 0
	e = mft.Buf.ReadIndex(0,r)
	if e!=nil { return nil,e }
	e = binary.Read(mft.Buf,binary.BigEndian,mft.Head)
	if e!=nil { return nil,e }
	mft.EntriesPerBlock = blockSize / MFTE_SIZE
	mft.Size = mft.Head.Num_BLK * mft.EntriesPerBlock
	
	return mft,nil
}
func (m* MFT) SaveMFTH() error {
	m.Buf.Pos = 0
	m.Size = m.Head.Num_BLK * m.EntriesPerBlock
	e := binary.Write(m.Buf,binary.BigEndian,m.Head)
	if e!=nil { return e }
	return m.Buf.WriteIndex(0,m.Range)
}
func (m* MFT) ClearMFT() {
	mfte := new(MFTE)
	for i:=uint32(1); i<m.Size; i++ {
		m.PutEntryLL(i,mfte)
	}
}
func (m* MFT) CreateEntry(i uint32) *MFTE {
	f := new(MFTE)
	f.File_IDX  = i
	f.File_MFT  = m.Head.MFT_ID
	f.Next_IDX  = 0
	f.First_IDX = i
	return f
}
func (m* MFT) GetEntry(i uint32) (*MFTE,error) {
	f,e := m.GetEntryLL(i)
	if e!=nil { return nil,e }
	if f.File_IDX!=i { return nil,cormfte }
	if f.File_MFT!=m.Head.MFT_ID { return nil,cormfte }
	return f,nil
}
func (m* MFT) GetEntryLL(i uint32) (*MFTE,error) {
	mer,ok := m.entry_cache.Get(i)
	if ok { return mer.(*MFTE),nil }
	mfe,e := m.GetEntryLLL(i)
	if e==nil {
		m.entry_cache.Add(i,mfe)
	}
	return mfe,e
}
func (m* MFT) GetEntryLLL(i uint32) (*MFTE,error) {
	if i==0 { return nil,badmfte }
	if i>=m.Size { return nil,badmfte }
	e := m.Buf.ReadIndex(int64(i),m.Range)
	if e!=nil { return nil,e }
	mfte := new(MFTE)
	e = binary.Read(m.Buf,binary.BigEndian,mfte)
	return mfte,e
}
/*
func (m* MFT) PutEntry(mfte *MFTE) error {
	return m.PutEntryLL(mfte.File_IDX,mfte)
}
*/
func (m* MFT) PutEntryLL(i uint32,mfte *MFTE) error {
	e := m.PutEntryLLL(i,mfte)
	if e==nil {
		m.entry_cache.Add(i,mfte)
	}
	return e
}
func (m* MFT) PutEntryLLL(i uint32,mfte *MFTE) error {
	if i==0 { return badmfte }
	if i>=m.Size { return badmfte }
	m.Buf.Pos = 0
	e := binary.Write(m.Buf,binary.BigEndian,mfte)
	if e!=nil { return e }
	e = m.Buf.WriteIndex(int64(i),m.Range)
	return e
}
func (m* MFT) buildChain(mfte *MFTE, chain *MFTE_Chain) error {
	idx := mfte.File_IDX
	off := uint64(0)
	var e error
	for {
		chain.Indeces = append(growarray_u32(chain.Indeces),mfte.File_IDX)
		chain.Off_BLK = append(growarray_u64(chain.Off_BLK),off)
		if mfte.End_BLK > mfte.Begin_BLK {
			off += mfte.End_BLK-mfte.Begin_BLK
		}
		if mfte.Next_IDX==0 { break }
		mfte,e = m.GetEntry(mfte.Next_IDX)
		if e!=nil { return e }
		if mfte.First_IDX!=idx { return cecmfte }
	}
	chain.TotalBLK = off
	return nil
}
func (m* MFT) GetEntryChainLL(i uint32) (*MFTE_Chain,error) {
	mfte,e := m.GetEntry(i)
	if e!=nil { return nil,e }
	if mfte.File_IDX!=mfte.First_IDX { return nil,mftenohead }
	chain := new(MFTE_Chain)
	e = m.buildChain(mfte,chain)
	if e!=nil { return nil,e }
	return chain,nil
}
func (m* MFT) GetEntryChain(i uint32) (*MFTE_Chain,error) {
	chainraw,ok := m.list_cache.Get(i)
	if ok { return chainraw.(*MFTE_Chain),nil }
	chain,e := m.GetEntryChainLL(i)
	if e!=nil {
		m.list_cache.Add(i,chain)
	}
	return chain,e
}
func (m* MFT) ResetEntryChain(i uint32){
	m.list_cache.Remove(i)
}
func (m* MFT) Allocate() (*MFTE,error) {
	for i:=uint32(0); i<m.Size; i++ {
		_,e := m.GetEntry(i)
		if e==cormfte { return m.CreateEntry(i),nil }
	}
	return nil,allocmfte
}

func MFT_IsAllocFail(e error) bool {
	return e==allocmfte
}

type MMFT struct{
	Mutex sync.Mutex
	MftByID map[uint32]*MFT
}
func (mm* MMFT) Init(){
	mm.MftByID = make(map[uint32]*MFT)
}
func (mm* MMFT) get(ii uint32) (*MFT,bool) {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()
	m,o := mm.MftByID[ii]
	return m,o
}
func (mm* MMFT) RandomGet() (uint32,bool) {
	n := rand.Uint32()
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()
	ok := false
	r := uint32(0)
	for i := range mm.MftByID {
		if !ok {
			ok = true
			r = i
		} else if (r^n)>(i^n) {
			r = i
		}
	}
	return r,ok
}
func (mm* MMFT) Set(m *MFT) {
	mm.Mutex.Lock()
	defer mm.Mutex.Unlock()
	mm.MftByID[m.Head.MFT_ID] = m
}
func (mm* MMFT) PutEntry(mfte *MFTE) error {
	return mm.PutEntryLL(mfte.File_MFT,mfte.File_IDX,mfte)
}
func (mm* MMFT) PutEntryLL(ii, i uint32,mfte *MFTE) error {
	m,ok := mm.get(ii)
	if !ok { return nomfte }
	return m.PutEntryLL(i,mfte)
}
func (mm* MMFT) GetEntry(ii, i uint32) (*MFTE,error) {
	f,e := mm.GetEntryLL(ii,i)
	if e!=nil { return nil,e }
	if f.File_IDX!=i { return nil,cormfte }
	if f.File_MFT!=ii { return nil,cormfte }
	return f,nil
}
func (mm* MMFT) GetEntryLL(ii, i uint32) (*MFTE,error) {
	m,ok := mm.get(ii)
	if !ok { return nil,nomfte }
	return m.GetEntryLL(i)
}
func (mm* MMFT) GetEntryChain(ii, i uint32) (*MFTE_Chain,error) {
	m,ok := mm.get(ii)
	if !ok { return nil,nomfte }
	return m.GetEntryChain(i)
}
func (mm* MMFT) ResetEntryChain(ii,i uint32) {
	m,ok := mm.get(ii)
	if ok { m.ResetEntryChain(i) }
}
func (mm* MMFT) CreateEntry(ii,i uint32) *MFTE {
	m,ok := mm.get(ii)
	if !ok { return nil }
	return m.CreateEntry(i)
}
func (mm* MMFT) Allocate(ii uint32) (*MFTE,error) {
	m,ok := mm.get(ii)
	if !ok { return nil,nomfte }
	return m.Allocate()
}


