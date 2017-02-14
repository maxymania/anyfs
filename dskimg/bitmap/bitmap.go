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
package bitmap

import "io"

func ScanRange(buf []byte,begin, end uint64) uint64 {
	for i := begin; i<end; i++ {
		bp := int(i>>3)
		bi := uint(i&7)
		if (buf[bp] & (1<<bi)) != 0 { return i }
	}
	return end
}
func ScanSetRange(buf []byte,begin, end uint64) uint64 {
	for i := begin; i<end; i++ {
		bp := int(i>>3)
		bi := uint(i&7)
		if (buf[bp] & (1<<bi)) == 0 { return i }
	}
	return end
}
func SetRange(buf []byte,begin, end uint64) uint64 {
	for i := begin; i<end; i++ {
		bp := int(i>>3)
		bi := uint(i&7)
		buf[bp] |= (1<<bi)
	}
	return end
}
func AllocRange(buf []byte,begin, end uint64) uint64 {
	for i := begin; i<end; i++ {
		bp := int(i>>3)
		bi := uint(i&7)
		if (buf[bp] & (1<<bi)) != 0 { return i }
		buf[bp] |= (1<<bi)
	}
	return end
}
func FreeRange(buf []byte,begin, end uint64) uint64 {
	for i := begin; i<end; i++ {
		bp := int(i>>3)
		bi := uint(i&7)
		buf[bp] &= ^(1<<bi)
	}
	return end
}

type RangeFunc func(buf []byte,begin, end uint64) uint64

type RAS interface{
	io.ReaderAt
	io.WriterAt
}

type BitRegion struct{
	Image  RAS
}
func (r *BitRegion) Apply(buf []byte,begin, end uint64,rf RangeFunc,write bool) (uint64,error) {
	p := int64(begin>>3)
	n := int64(end>>3)+1
	off := uint64(p)<<3
	if int64(len(buf))>n {
		buf = buf[:int(n)]
	} else if int64(len(buf))<n {
		n = int64(len(buf))
		end = (uint64(n)<<3)|7
	}
	n2,e := r.Image.ReadAt(buf,p)
	if e!=nil { return begin,e }
	if int64(n2)<n { buf = buf[:n2] }
	res := rf(buf,begin-off,end-off)+off
	if write {
		_,e := r.Image.WriteAt(buf,p)
		return res,e
	}
	return res,nil
}

