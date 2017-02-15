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

type FixedIO struct{
	Buffer []byte
	Pos    int
}
func (f* FixedIO) ReadIndex(i int64,absr io.ReaderAt) error {
	n,e := absr.ReadAt(f.Buffer,int64(len(f.Buffer))*i)
	if len(f.Buffer)==n { e = nil }
	f.Pos = 0
	return e
}
func (f* FixedIO) WriteIndex(i int64,absr io.WriterAt) error {
	n,e := absr.WriteAt(f.Buffer,int64(len(f.Buffer))*i)
	if len(f.Buffer)==n { e = nil }
	f.Pos = 0
	return e
}
func (f* FixedIO) Write(p []byte) (n int, err error) {
	b := f.Pos
	e := b+len(p)
	if e>len(f.Buffer) { err = io.EOF; e = len(f.Buffer) }
	n = e-b
	copy(f.Buffer[b:],p[:n])
	f.Pos = e
	return
}
func (f* FixedIO) Read(p []byte) (n int, err error) {
	b := f.Pos
	e := b+len(p)
	if e>len(f.Buffer) { err = io.EOF; e = len(f.Buffer) }
	n = e-b
	copy(p[:n],f.Buffer[b:])
	f.Pos = e
	return
}

