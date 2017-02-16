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

const Superblock_MagicNumber = 0x19771025

type Superblock struct{
	MagicNumber uint32
	BlockSize   uint32
	DiskSerial  uint64
	Block_Len   uint64
	Bitmap_BLK  uint64
	Bitmap_LEN  uint64
	FirstMFT    uint64
	DirSegSize  uint32 /* Directory Segment Size */
}

func (sb *Superblock) LoadSuperblock(i int64,rwa dskimg.IoReaderWriterAt) error{
	fio := &dskimg.FixedIO{make([]byte,256),0}
	_,e := rwa.ReadAt(fio.Buffer,i)
	if e!=nil { return e }
	return binary.Read(fio,binary.BigEndian,sb)
}
func (sb *Superblock) StoreSuperblock(i int64,rwa dskimg.IoReaderWriterAt) error{
	fio := &dskimg.FixedIO{make([]byte,256),0}
	e := binary.Write(fio,binary.BigEndian,sb)
	if e!=nil { return e }
	_,e = rwa.WriteAt(fio.Buffer,i)
	return e
}
func (sb *Superblock) Offset(i uint64) int64 {
	return int64(i*uint64(sb.BlockSize))
}
func (sb *Superblock) Length(i uint64) int64 {
	return int64(i*uint64(sb.BlockSize))
}

