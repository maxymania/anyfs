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

package main

import "os"
//import "github.com/maxymania/anyfs/dskimg/bitmap"
import "github.com/maxymania/anyfs/dskimg/fs1"
//import _ "github.com/maxymania/anyfs/dskimg/ods"
import "fmt"
import "flag"
import dbgpkg "github.com/maxymania/anyfs/debug"

const (
	BZ_0 = 1<<iota
	BZ_1
	BZ_2
	BZ_3
	BZ_4
	BZ_5
	BZ_6
	BZ_7
	BZ_8
	BZ_9
)

var image = flag.String("image", "", "The file-system image to be formatted")
var bzk = flag.Int("bsize", 4, fmt.Sprint("block size (in kb) (valid is ",BZ_0,BZ_1,BZ_2,BZ_3,BZ_4,BZ_5,BZ_6,BZ_7,BZ_8,BZ_9,")"))
var bom = flag.String("bsord", "K", "M = 'block size in MB instead of KB';  * = 'block size in byte instead of KB'")

var mzk = flag.Int("mft", 4, fmt.Sprint("mft size (in kb) (valid is ",BZ_0,BZ_1,BZ_2,BZ_3,BZ_4,BZ_5,BZ_6,BZ_7,BZ_8,BZ_9,")"))
var mom = flag.String("mftord", "K", "M = 'mft size in MB instead of KB';  * = 'mft size in byte instead of KB'")

var offset = flag.Int("sbo",512,"Superblock Offset")

var trace = flag.Bool("trace", false, "print deep tracing messages")

func main(){
	flag.Parse()
	dbgpkg.TraceOn = *trace
	if *image=="" {
		flag.PrintDefaults()
		return
	}
	f,e := os.OpenFile(*image,os.O_RDWR,0666) // 
	if e!=nil {
		fmt.Println("Error: ",e)
		flag.PrintDefaults()
		return
	}
	defer f.Close()
	mkfs := new(fs1.MkfsInfo)
	mkfs.BlockSize = uint32(*bzk)
	switch mkfs.BlockSize {
	case BZ_0,BZ_1,BZ_2,BZ_3,BZ_4,BZ_5,BZ_6,BZ_7,BZ_8,BZ_9:
		if *bom=="K" {
			mkfs.BlockSize<<=10
		} else if *bom=="M" {
			mkfs.BlockSize<<=20
		}
	default:
		flag.PrintDefaults()
		return
	}
	mkfs.MftBlocks = uint32(*mzk)
	switch mkfs.MftBlocks {
	case BZ_0,BZ_1,BZ_2,BZ_3,BZ_4,BZ_5,BZ_6,BZ_7,BZ_8,BZ_9:
		if *bom=="K" {
			mkfs.MftBlocks<<=10
		} else if *bom=="M" {
			mkfs.MftBlocks<<=20
		}
	default:
		flag.PrintDefaults()
		return
	}
	mkfs.MftBlocks += mkfs.BlockSize-1
	mkfs.MftBlocks /= mkfs.BlockSize
	fs := new(fs1.FileSystem)
	fs.Device = f
	fs.NoSync = true /* We don't need auto-FSYNC */
	err := fs.Mkfs(int64(*offset),mkfs)
	if err!=nil {
		fmt.Println("Error: ",err)
		flag.PrintDefaults()
		return
	}
}

