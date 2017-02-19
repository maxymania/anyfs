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

import "sync"
import "github.com/maxymania/anyfs/dskimg/ods"
import "github.com/maxymania/anyfs/security"
import "time"

type MetaDataFile struct{
	Backing *File
	Memory  *ods.MetaDataMemory
	Dirty   bool
	DirtySync sync.Mutex
}

func (m *MetaDataFile) init(fs *FileSystem, ii, i uint32) error{
	m.Backing = &File{fs,ii,i}
	m.Memory  = new(ods.MetaDataMemory)
	m.Memory.Init()
	sz,err := m.Backing.Size()
	if err!=nil { return err }
	m.Memory.LoadMax(m.Backing,sz)
	m.Dirty = false
	return nil
}
func (m *MetaDataFile) flush() {
	m.DirtySync.Lock()
	defer m.DirtySync.Unlock()
	if !m.Dirty { return }
	m.Memory.SerializeTime(m.Backing)
	m.Dirty = false
}
func (m *MetaDataFile) initialContent() {
	full_control := security.PrFullControl.AllowVector()
	tm := time.Now()
	m.Memory.BirthTimeSet(tm)
	m.Memory.WriteTimeSet(tm)
	m.Memory.AccessTimeSet(tm)
	m.Memory.SerializeTime(m.Backing)
	m.Memory.PutAcl(security.AccessControlEntry{security.SIDC_SYSTEM  ,full_control},m.Backing)
	m.Memory.PutAcl(security.AccessControlEntry{security.SIDC_ROOT    ,full_control},m.Backing)
}
func (m *MetaDataFile) PutAcl(ace security.AccessControlEntry) {
	m.Memory.PutAcl(ace,m.Backing)
}


