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
package xmd

import (
	"time"
	"syscall"
	"encoding/binary"
	"encoding/json"
	"path/filepath"
)

func encodeTime(ts time.Time) []byte{
	b := make([]byte,12)
	sec := ts.Unix()
	nsec := ts.UnixNano()
	binary.BigEndian.PutUint64(b,uint64(sec))
	binary.BigEndian.PutUint32(b[8:],uint32(nsec))
	return b
}

func decodeTime(attr []byte) time.Time{
	sec := int64(binary.BigEndian.Uint64(attr))
	nsec := int64(int32(binary.BigEndian.Uint32(attr[8:])))
	return time.Unix(sec,nsec).UTC()
}

type userInfoJson struct{
	SYS_create    time.Time `json:"create"`
	SYS_metadata  time.Time `json:"metadata"`
	SYS_name      time.Time `json:"name"`
	SYS_writedata time.Time `json:"writedata"`
	SYS_readdata  time.Time `json:"readdata"`
	SYS_close     time.Time `json:"close"`
	SYS_open      time.Time `json:"open"`
	SYS_addchild  time.Time `json:"addchild"`
	SYS_rmchild   time.Time `json:"rmchild"`
}

func extract(fn string) []byte{
	i := &userInfoJson{}
	b := make([]byte,12)
	syscall.Getxattr(fn,"user.SYS_create",b)
	i.SYS_create = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_metadata",b)
	i.SYS_metadata = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_name",b)
	i.SYS_name = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_writedata",b)
	i.SYS_writedata = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_readdata",b)
	i.SYS_readdata = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_close",b)
	i.SYS_close = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_open",b)
	i.SYS_open = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_addchild",b)
	i.SYS_addchild = decodeTime(b)
	syscall.Getxattr(fn,"user.SYS_rmchild",b)
	i.SYS_rmchild = decodeTime(b)
	attr,_ := json.Marshal(i)
	return attr
}

func updateMetadata(fn string){
	syscall.Setxattr(fn,"user.SYS_metadata",encodeTime(time.Now()),0)
}

func createFile(fn string){
	now := encodeTime(time.Now())
	syscall.Setxattr(fn,"user.SYS_create",now,0)
	
	syscall.Setxattr(fn,"user.SYS_metadata",now,0)
	syscall.Setxattr(fn,"user.SYS_name",now,0)
	syscall.Setxattr(fn,"user.SYS_writedata",now,0)
	syscall.Setxattr(fn,"user.SYS_readdata",now,0)
	syscall.Setxattr(fn,"user.SYS_close",now,0)
	syscall.Setxattr(fn,"user.SYS_open",now,0)
	syscall.Setxattr(fn,"user.SYS_addchild",now,0)
	syscall.Setxattr(fn,"user.SYS_rmchild",now,0)
	
	dn := filepath.Dir(fn)
	syscall.Setxattr(dn,"user.SYS_addchild",now,0)
}

func linkFile(fn string){
	syscall.Setxattr(fn,"user.SYS_name",encodeTime(time.Now()),0)
}

func fakeDeleteFile(fn string){
	dn := filepath.Dir(fn)
	syscall.Setxattr(dn,"user.SYS_rmchild",encodeTime(time.Now()),0)
}

func deleteFile(fn string){
	fakeDeleteFile(fn)
	now := encodeTime(time.Now())
	syscall.Setxattr(fn,"user.SYS_delete",now,0)
	syscall.Setxattr(fn,"user.SYS_name",now,0)
}

func renameFile(fn string){
	now := encodeTime(time.Now())
	syscall.Setxattr(fn,"user.SYS_name",now,0)
	dn := filepath.Dir(fn)
	syscall.Setxattr(dn,"user.SYS_addchild",now,0)
}

func openFile(fn string){
	syscall.Setxattr(fn,"user.SYS_open",encodeTime(time.Now()),0)
}

func closeFile(fn string){
	syscall.Setxattr(fn,"user.SYS_close",encodeTime(time.Now()),0)
}

func writeFile(fn string){
	syscall.Setxattr(fn,"user.SYS_writedata",encodeTime(time.Now()),0)
}

func readFile(fn string){
	syscall.Setxattr(fn,"user.SYS_readdata",encodeTime(time.Now()),0)
}

