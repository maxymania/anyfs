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
package fs1

import "os"

import "github.com/maxymania/anyfs/dskimg"
import "github.com/maxymania/anyfs/dskimg/ods"
import "github.com/maxymania/anyfs/dskimg/bitmap"
import "errors"
import "sync"

var badmz = errors.New("Bad Magic number")

var oor = errors.New("Out of Resources")

const (
	FS_SPECIAL_ROOT = 1+iota
)

type MkfsInfo struct{
	BlockSize  uint32
	MftBlocks  uint32
}
func (mf *MkfsInfo) bitmap(i int64, blknum uint64) (blk,lng uint64){
	blk  = uint64(i+256)+uint64(mf.BlockSize)-1
	blk /= uint64(mf.BlockSize)
	lng  = ((blknum+7)>>3)+uint64(mf.BlockSize)-1
	lng /= uint64(mf.BlockSize)
	return
}


type FileSystem struct{
	Device *os.File
	SB     *ods.Superblock
	MMFT    ods.MMFT
	MFTLck  sync.Mutex
	BitMap  bitmap.BitRegion
	BMLck   sync.Mutex
	Temp    uint32
}

func (f *FileSystem) Mkfs(i int64, mf *MkfsInfo) error {
	fif,e := f.Device.Stat()
	if e!=nil { return e }
	f.SB = new(ods.Superblock)
	f.SB.MagicNumber = ods.Superblock_MagicNumber
	f.SB.BlockSize   = mf.BlockSize
	f.SB.DiskSerial  = 12345
	f.SB.Block_Len   = uint64(fif.Size())/uint64(mf.BlockSize)
	f.SB.Bitmap_BLK,f.SB.Bitmap_LEN = mf.bitmap(i,f.SB.Block_Len)
	
	img := dskimg.NewSectionIo(f.Device,f.SB.Offset(f.SB.Bitmap_BLK),f.SB.Length(f.SB.Bitmap_LEN))
	buffer := make([]byte,int(mf.BlockSize))
	
	img.Overwrite(buffer)
	f.BitMap.Image = img
	endbm := f.SB.Bitmap_BLK+f.SB.Bitmap_LEN
	
	f.SB.FirstMFT = endbm
	
	mftblocks := uint64(mf.MftBlocks)
	_,e = f.BitMap.Apply(buffer,0,endbm+mftblocks,bitmap.SetRange,true)
	if e!=nil { return e }
	
	mft,e := ods.NewMFT(dskimg.NewSectionIo(f.Device,f.SB.Offset(endbm),f.SB.Length(mftblocks)),mf.BlockSize)
	if e!=nil { return e }
	mft.Head.MFT_ID  = 1234
	mft.Head.Num_BLK = mf.MftBlocks
	mft.Head.NextMFT = 0
	e = mft.SaveMFTH()
	if e!=nil { return e }
	mft.ClearMFT()
	
	f.MMFT.Init()
	f.MMFT.Set(mft)
	
	f.Temp = mft.Head.MFT_ID
	
	{
		mfte := f.MMFT.CreateEntry(f.Temp,FS_SPECIAL_ROOT)
		mfte.FileType = ods.FT_DIR
		
		e := f.MMFT.PutEntry(mfte)
		if e!=nil { return e }
	}
	
	return f.SB.StoreSuperblock(i,f.Device)
}
func (f *FileSystem) LoadFileSystem(i int64) error {
	f.SB = new(ods.Superblock)
	e := f.SB.LoadSuperblock(i,f.Device)
	if e!=nil { return e }
	if f.SB.MagicNumber != ods.Superblock_MagicNumber { return badmz }
	f.BitMap.Image = dskimg.NewSectionIo(f.Device,f.SB.Offset(f.SB.Bitmap_BLK),f.SB.Length(f.SB.Bitmap_LEN))
	
	img := dskimg.NewSectionIo(f.Device,f.SB.Offset(f.SB.FirstMFT),f.SB.Length(1))
	mft,e := ods.NewMFT(img,f.SB.BlockSize)
	if e!=nil { return e }
	
	img.SetSectionSize(f.SB.Length(uint64(mft.Head.Num_BLK)))
	
	f.MMFT.Init()
	f.MMFT.Set(mft)
	
	f.Temp = mft.Head.MFT_ID
	
	return nil
}

// Get file.
func (f *FileSystem) GetFile(ii, i uint32) *File {
	return &File{f,ii,i}
}

// Get Root directory of this FS.
func (f *FileSystem) GetRootDir() *File {
	return &File{f,f.Temp,FS_SPECIAL_ROOT}
}

/*
 * Creates a new File in filesystem.
 * 'ft' must be one of FT_FILE, FT_DIR, FT_FIFO
 */
func (f *FileSystem) CreateFile(ft uint8) (*File,error) {
	f.MFTLck.Lock()
	defer f.MFTLck.Unlock()
	retries := 32
	id,ok := f.MMFT.RandomGet()
	if !ok { return nil,oor }
	mfte,e := f.MMFT.Allocate(id)
	for ods.MFT_IsAllocFail(e) {
		if retries<1 { return nil,e }
		retries--
		id,ok = f.MMFT.RandomGet()
		if !ok { return nil,oor }
		mfte,e = f.MMFT.Allocate(id)
	}
	if e!=nil { return nil,e }
	mfte.FileType = ft
	e = f.MMFT.PutEntry(mfte)
	if e!=nil { return nil,e }
	return &File{f,mfte.File_MFT,mfte.File_IDX},nil
}


