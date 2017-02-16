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

import (
	"flag"
	"fmt"
	"os"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	
	"github.com/maxymania/anyfs/dskimg/fs1"
	"github.com/maxymania/anyfs/dskimg/fs1drv"
)

var debug = flag.Bool("debug", false, "print debugging messages.")
var image = flag.String("image", "", "The file-system image to be formatted")
var mount = flag.String("mount", "", "Mount-Point")
var offset = flag.Int("sbo",512,"Superblock Offset")

func main() {
	// Scans the arg list and sets up flags
	
	flag.Parse()
	if (*image=="") || (*mount=="") {
		flag.PrintDefaults()
		return
	}
	f,e := os.OpenFile(*image,os.O_RDWR,0666) // 
	if e!=nil {
		fmt.Println("Error: ",e)
		flag.PrintDefaults()
		return
	}
	fs := new(fs1.FileSystem)
	fs.Device = f
	e = fs.LoadFileSystem(int64(*offset))
	if e!=nil {
		fmt.Println("Error: ",e)
		flag.PrintDefaults()
		return
	}
	rd := fs.GetRootDir()
	rdir,e := rd.AsDirectory()
	if e!=nil {
		fmt.Println("Error: ",e)
		flag.PrintDefaults()
		return
	}
	
	root := new(fs1drv.DirNode)
	root.Node = nodefs.NewDefaultNode()
	root.Backing = rd
	root.Dir = rdir
	
	conn := nodefs.NewFileSystemConnector(root, nil)
	server, err := fuse.NewServer(conn.RawFS(), *mount, &fuse.MountOptions{
		Debug: *debug,
	})
	if err != nil {
		fmt.Printf("Mount fail: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Mounted!")
	server.Serve()
}

