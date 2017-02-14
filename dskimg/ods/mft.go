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

import "encoding/binary"
import "github.com/maxymania/anyfs/dskimg"
import "errors"

var badmfte = errors.New("Bad MFT index")

const MFTE_SIZE = 64

type MFTH struct{
	MFT_ID    uint32
	Num_BLK   uint64
	NextMFT   uint64
}

type MFTE struct{
	File_MFT  uint32
	File_IDX  uint32
	Cookie    uint64
	Begin_BLK uint64
	End_BLK   uint64
	Next_IDX  uint32
	TypeFlag  uint32
}

type MFT struct{
	Head  *MFTH
	Range RAS
	Buf   *dskimg.FixedIO
}
func NewMFT(r RAS) (*MFT,error) {
	mft := new(MFT)
	mft.Head = new(MFTH)
	mft.Range = r
	mft.Buf = &dskimg.FixedIO{make([]byte,MFTE_SIZE),0}
	e := mft.Buf.ReadIndex(0,r)
	if e!=nil { return nil,e }
	e = binary.Read(mft.Buf,binary.BigEndian,mft.Head)
	if e!=nil { return nil,e }
	return mft,nil
}
func (m* MFT) GetEntry(i uint32) (*MFTE,error) {
	if i==0 { return nil,badmfte }
	e := m.Buf.ReadIndex(int64(i),m.Range)
	if e!=nil { return nil,e }
	mfte := new(MFTE)
	e = binary.Read(m.Buf,binary.BigEndian,mfte)
	return mfte,e
}
func (m* MFT) PutEntry(i uint32,mfte *MFTE) error {
	if i==0 { return badmfte }
	e := binary.Write(m.Buf,binary.BigEndian,mfte)
	if e!=nil { return e }
	e = m.Buf.WriteIndex(int64(i),m.Range)
	return e
}


