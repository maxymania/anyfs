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

package security

import "fmt"

type SID uint64

const (
	SID_UID  = 0x11a10000
	SID_GID  = 0x11b10000
	SID_TYPE = 0x22000000
)

func LoweUpperSID(low,up uint32) SID {
	return SID((uint64(low)<<32)|uint64(up))
}
func GidSID(gid uint32) SID {
	return SID(uint64(SID_GID<<32)|uint64(gid))
}
func UidSID(uid uint32) SID {
	return SID(uint64(SID_UID<<32)|uint64(uid))
}
func TypeSID(ty uint32) SID {
	return SID(uint64(SID_TYPE<<32)|uint64(ty))
}

func (s SID) Upper() uint32 {
	return uint32(uint64(s)>>32)
}
func (s SID) Lower() uint32 {
	return uint32(uint64(s)&0xffffffff)
}
func (s SID) Gid() uint32 {
	if s.Upper()!= SID_GID { return ^uint32(0) }
	return s.Lower()
}
func (s SID) Uid() uint32 {
	if s.Upper()!= SID_UID { return ^uint32(0) }
	return s.Lower()
}
func (s SID) String() string{
	switch s.Upper() {
	case SID_UID:
		return fmt.Sprint("uid:",s.Lower())
	case SID_GID:
		return fmt.Sprint("gid:",s.Lower())
	case SID_TYPE:
		return fmt.Sprint("type:",s.Lower())
	}
	return fmt.Sprint("SID:",s.Upper(),"-",s.Lower())
}

