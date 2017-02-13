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
package lcr

import (
	"io"
	"time"
)

type TypeID int
const (
	BLOB = TypeID(iota)
	STRING
	INTEGER
	DATETIME
)

type StoragePolicy string
const (
	STORE_HOT    = StoragePolicy("standard.policy.Hot")
	STORE_MEDIUM = StoragePolicy("standard.policy.Medium")
	STORE_COLD   = StoragePolicy("standard.policy.Cold")
)

type Blob interface{
	io.ReaderAt
	io.WriterAt
	io.Closer
	Truncate(i int64) error
	Length() int64
	UnderlyingObject() interface{}
}

type Values interface{
	GetBlob(i int) Blob
	GetString(i int) string
	GetInt(i int) int64
	GetDate(i int) time.Time
	
	GetTypeID(i int) TypeID
	Length() int
}

type Node interface{
	GetProperty(name string) Values
	PutProperties(name string, overwrite bool, i ...interface{}) Values
	Lookup(name string) Node
	Create(name string) Node
}

