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
package dskimg

import "io"

type IoReaderWriterAt interface{
	io.ReaderAt
	io.WriterAt
}

type SectionIo struct{
	rwa IoReaderWriterAt
	base int64
	lngt int64
}

func NewSectionIo(rwa IoReaderWriterAt, base int64, length int64) *SectionIo { return &SectionIo{rwa,base,length} }

func (s *SectionIo) Overwrite(buf []byte) {
	z := s.Size()
	p := int64(0)
	n := int64(len(buf))
	for p<z {
		s.WriteAt(buf,p)
		p += n
	}
}
func (s *SectionIo) SetSectionSize(length int64) { s.lngt = length }
func (s *SectionIo) WriteAt(p []byte, off int64) (n int, err error) {
	if off<0 { return 0, io.EOF }
	if off>=s.lngt { return 0,io.EOF }
	
	if max := s.lngt-off; max<int64(len(p)) {
		n,err = s.rwa.WriteAt(p[:int(max)],off+s.base)
		if err==nil { err = io.EOF }
	}else{
		n,err = s.rwa.WriteAt(p,off+s.base)
	}
	return
}
func (s *SectionIo) ReadAt(p []byte, off int64) (n int, err error) {
	if off<0 { return 0, io.EOF }
	if off>=s.lngt { return 0,io.EOF }
	
	if max := s.lngt-off; max<int64(len(p)) {
		n,err = s.rwa.ReadAt(p[:int(max)],off+s.base)
		if err==nil { err = io.EOF }
	}else{
		n,err = s.rwa.ReadAt(p,off+s.base)
	}
	return
}
func (s *SectionIo) Size() int64 { return s.lngt }
