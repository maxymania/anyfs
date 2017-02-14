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

var badmfte = errors.New("Bad MFT index")
var cormfte = errors.New("Corrupted MFT Entry")

var nomfte = errors.New("No such MFT")

const MFTE_SIZE = 64

type MFTH struct{
	MFT_ID    uint32
	Num_BLK   uint32
	NextMFT   uint64
}

type MFTE struct{
	File_MFT  uint32
	File_IDX  uint32
	Cookie    uint64
	Begin_BLK uint64
	End_BLK   uint64
	Next_IDX  uint32 /* Next Element in Chain. */
	First_IDX uint32 /* First Element in Chain. */
	TypeFlag  uint32
}

type MFT struct{
	Head  *MFTH
	Range RAS
	Buf   *dskimg.FixedIO
	Size  uint32
	EntriesPerBlock uint32
}
func NewMFT(r RAS,blockSize uint32) (*MFT,error) {
	mft := new(MFT)
	mft.Head = new(MFTH)
	mft.Range = r
	mft.Buf = &dskimg.FixedIO{make([]byte,MFTE_SIZE),0}
	mft.Size = 0
	e := mft.Buf.ReadIndex(0,r)
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
	if i==0 { return nil,badmfte }
	if i>=m.Size { return nil,badmfte }
	e := m.Buf.ReadIndex(int64(i),m.Range)
	if e!=nil { return nil,e }
	mfte := new(MFTE)
	e = binary.Read(m.Buf,binary.BigEndian,mfte)
	return mfte,e
}
func (m* MFT) PutEntry(mfte *MFTE) error {
	return m.PutEntryLL(mfte.File_IDX,mfte)
}
func (m* MFT) PutEntryLL(i uint32,mfte *MFTE) error {
	if i==0 { return badmfte }
	if i>=m.Size { return badmfte }
	e := binary.Write(m.Buf,binary.BigEndian,mfte)
	if e!=nil { return e }
	e = m.Buf.WriteIndex(int64(i),m.Range)
	return e
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

