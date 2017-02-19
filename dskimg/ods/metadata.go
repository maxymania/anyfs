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

import "sync"
import "encoding/binary"
import "github.com/maxymania/anyfs/dskimg"

import "github.com/maxymania/anyfs/security"

import "time"

const (
	MDE_Free = 0xa0+iota
	MDE_BirthTime
	MDE_WriteTime
	MDE_AccessTime
	MDE_ACE
)

type MetaDataEntry struct {
	Type  uint8
	Data1 uint8
	Data2 uint16
	Data3 uint32
	Data4 uint64
}
func (m *MetaDataEntry) put(buf *dskimg.FixedIO) error{
	return binary.Write(buf, binary.BigEndian, m)
}
func (m *MetaDataEntry) get(buf *dskimg.FixedIO) error{
	return binary.Read(buf, binary.BigEndian, m)
}

type metaDataTime struct {
	idx    int64
	tstamp *time.Time
}
func (m *metaDataTime) Set(tm time.Time) {
	ntm := new(time.Time)
	*ntm = tm
	m.tstamp = ntm
}
func (m *metaDataTime) fromMDE(mde *MetaDataEntry) {
	ntm := new(time.Time)
	*ntm = time.Unix(int64(mde.Data4),int64(mde.Data3))
	m.tstamp = ntm
}
func (m *metaDataTime) toMDE(Type  uint8,mde *MetaDataEntry) (*MetaDataEntry){
	if m.tstamp==nil { return nil }
	uts := uint64(m.tstamp.Unix())
	utsns := uint32(m.tstamp.UnixNano())
	*mde = MetaDataEntry{Type,0,0,utsns,uts}
	return mde
}


type MetaDataMemory struct {
	ACL  security.AccessControlList
	mutex     sync.Mutex
	buf       *dskimg.FixedIO
	birthTime metaDataTime
	writeTime metaDataTime
	accesTime metaDataTime
	aclidx    map[security.SID]int64
	freelist  []int64
	length    int64
}
func (m *MetaDataMemory) getNewIndex() int64 {
	i := m.length
	if len(m.freelist)>0 {
		i = m.freelist[0]
		m.freelist = m.freelist[1:]
	} else {
		m.length++
	}
	return i
}
func (m *MetaDataMemory) insertMDE(mde *MetaDataEntry,ras RAS) {
	i := m.getNewIndex()
	m.buf.Pos = 0
	mde.put(m.buf)
	m.buf.WriteIndex(i,ras)
}
func (m *MetaDataMemory) Init(){
	m.ACL  = make(security.AccessControlList)
	m.buf = &dskimg.FixedIO{make([]byte,16),0}
	m.aclidx  = make(map[security.SID]int64)
}

func (m *MetaDataMemory) BirthTime() *time.Time {
	return m.birthTime.tstamp
}
func (m *MetaDataMemory) BirthTimeSet(tm time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.birthTime.tstamp==nil { m.birthTime.idx = m.getNewIndex(); return }
	m.birthTime.Set(tm)
}

func (m *MetaDataMemory) WriteTime() *time.Time {
	return m.writeTime.tstamp
}
func (m *MetaDataMemory) WriteTimeSet(tm time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.birthTime.tstamp==nil { m.birthTime.idx = m.getNewIndex(); return }
	m.writeTime.Set(tm)
}

func (m *MetaDataMemory) AccessTime() *time.Time {
	return m.accesTime.tstamp
}
func (m *MetaDataMemory) AccessTimeSet(tm time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.birthTime.tstamp==nil { m.birthTime.idx = m.getNewIndex(); return }
	m.accesTime.Set(tm)
}

func (m *MetaDataMemory) SerializeTime(ras RAS) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	data := new(MetaDataEntry)
	
	m.buf.Pos = 0
	mde := m.birthTime.toMDE(MDE_BirthTime,data)
	if mde!=nil { mde.put(m.buf); m.buf.WriteIndex(m.birthTime.idx,ras) }
	
	m.buf.Pos = 0
	mde  = m.writeTime.toMDE(MDE_WriteTime,data)
	if mde!=nil { mde.put(m.buf); m.buf.WriteIndex(m.writeTime.idx,ras) }
	
	m.buf.Pos = 0
	mde  = m.accesTime.toMDE(MDE_AccessTime,data)
	if mde!=nil { mde.put(m.buf); m.buf.WriteIndex(m.accesTime.idx,ras) }
	
	return
}

func (m *MetaDataMemory) loadEntry(ras RAS, mde *MetaDataEntry, i int64) error{
	err := m.buf.ReadIndex(i,ras)
	if err!=nil { return err }
	mde.get(m.buf)
	switch mde.Type {
	case MDE_Free:
		m.freelist = append(m.freelist,i)
	case MDE_BirthTime:
		m.birthTime.fromMDE(mde)
	case MDE_WriteTime:
		m.writeTime.fromMDE(mde)
	case MDE_AccessTime:
		m.accesTime.fromMDE(mde)
	case MDE_ACE: {
		sid := security.SID(mde.Data4)
		acv := security.AccessControlVector(mde.Data3)
		m.ACL.AddEntry(security.AccessControlEntry{sid,acv})
		m.aclidx[sid]=i
		}
	}
	return nil
}
func (m *MetaDataMemory) Load(ras RAS) {
	mde := new(MetaDataEntry)
	i:=int64(0)
	for ; true; i++ {
		err := m.loadEntry(ras,mde,i)
		if err!=nil { break }
	}
	m.length = i
}
func (m *MetaDataMemory) LoadMax(ras RAS, max int64) {
	mde := new(MetaDataEntry)
	max /= 16
	for i:=int64(0); i<max; i++ {
		err := m.loadEntry(ras,mde,i)
		if err!=nil { continue }
	}
	m.length = max
}


func (m *MetaDataMemory) PutAcl(ace security.AccessControlEntry, ras RAS) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	i,ok := m.aclidx[ace.Subject]
	if !ok { i = m.getNewIndex() }
	m.ACL.AddEntry(ace)
	rights,_ := m.ACL[ace.Subject]
	mde := &MetaDataEntry{MDE_ACE,0,0,uint32(rights),uint64(ace.Subject)}
	mde.put(m.buf)
	m.buf.WriteIndex(i,ras)
}

