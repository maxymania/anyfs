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

type Privileges uint16

const (
	PrExecute = Privileges(1 << iota)
	PrReadData
	PrWriteData
	PrAppend
	PrDelete
	PrReadAttr
	PrWriteAttr
	PrReadPermissions
	PrWritePermissions
	PrLinkOrRename
	PrDeleteChilds
	
	/* These Constants Implement combination of the above flags. */
	PrRead   = PrReadData|PrReadAttr|PrReadPermissions
	PrWrite  = PrWriteData|PrWriteAttr|PrAppend
	PrReadAndExecute = PrRead|PrExecute
	PrModify = PrWrite|PrDelete|PrDeleteChilds|PrWritePermissions
	
	PrFullControl = PrExecute|PrReadData|PrWriteData|PrAppend|
			PrDelete|PrReadAttr|PrWriteAttr|PrReadPermissions|
			PrWritePermissions|PrLinkOrRename|PrDeleteChilds
	PrNone = Privileges(0)
)

type privilege_pair struct{
	name string
	priv Privileges
}
var privilege_pairs = [...]privilege_pair{
	privilege_pair{"PrFullControl",PrFullControl},
	privilege_pair{"PrModify",PrModify},
	privilege_pair{"PrReadAndExecute",PrReadAndExecute},
	privilege_pair{"PrWrite",PrWrite},
	privilege_pair{"PrRead",PrRead},
	
	privilege_pair{"PrExecute",PrExecute},
	privilege_pair{"PrReadData",PrReadData},
	privilege_pair{"PrWriteData",PrWriteData},
	privilege_pair{"PrAppend",PrAppend},
	privilege_pair{"PrDelete",PrDelete},
	privilege_pair{"PrReadAttr",PrReadAttr},
	privilege_pair{"PrWriteAttr",PrWriteAttr},
	privilege_pair{"PrReadPermissions",PrReadPermissions},
	privilege_pair{"PrWritePermissions",PrWritePermissions},
	privilege_pair{"PrLinkOrRename",PrLinkOrRename},
	privilege_pair{"PrDeleteChilds",PrDeleteChilds},
	
	privilege_pair{"PrNone",PrNone},
}
func (p Privileges) isMissing(full, has Privileges) bool{
	/* All 'p' bits must be set in 'full' */
	if (full&p)!=full { return false }
	/* No 'p' bit must be set in 'has' */
	if (has&full)!=Privileges(0) { return false }
	return true
}
func (p Privileges) asstr() string {
	if p==PrNone { return "" }
	return p.String()
}
func (p Privileges) String() string {
	s := ""
	//if p==PrNone { return "PrNone" }
	has := PrNone
	for _,pair := range privilege_pairs {
		if pair.priv==Privileges(0) { continue }
		if !p.isMissing(pair.priv,has) { continue }
		has |= pair.priv
		if s != "" { s += "," }
		s += pair.name
		if has==Privileges(0){ break }
	}
	if has!=p {
		return s+ " " +fmt.Sprintf("Privileges(%d)",int(uint32(p)))
	}
	if s=="" { return "PrNone" }
	return s
}
func PrivilegesFrom(s string) Privileges {
	for _,pair := range privilege_pairs {
		if pair.name != s { continue }
		return pair.priv
	}
	return PrNone
}

// Is 'p' a superset of 'n'?
func (p Privileges) Match(n Privileges) bool{
	return (n&p)==n
}

// p.IndexOf(n ...Privileges) = i  WHERE p.Match(n[i])
func (p Privileges) IndexOf(n ...Privileges) int{
	for i,pset := range n {
		if (pset&p)==pset { return i }
	}
	return -1
}

// p.MatchesOne(n ...Privileges) = p.IndexOf(n...)!= -1
func (p Privileges) MatchesOne(n ...Privileges) bool{
	for _,pset := range n {
		if (pset&p)==pset { return true }
	}
	return false
}


type AccessControlVector uint32

func (p Privileges) DenyVector() AccessControlVector {
	return AccessControlVector(uint32(uint16(p))<<16)
}
func (p Privileges) AllowVector() AccessControlVector {
	return AccessControlVector(uint32(uint16(p)))
}

func (a AccessControlVector) Deny() Privileges{
	b := uint32(a)>>16
	return Privileges(uint16(b))
}

func (a AccessControlVector) Allow() Privileges{
	b := uint32(a)&0xFFFF
	return Privileges(uint16(b))
}
func (a AccessControlVector) Effective() Privileges{
	return a.Allow()&^a.Deny()
}

func (a AccessControlVector) String() string {
	deny := a.Deny()
	allow := a.Allow()
	if deny==PrNone { return fmt.Sprint("Allow{",allow.asstr(),"}") }
	if allow==PrNone { return fmt.Sprint("Deny{",deny.asstr(),"}") }
	return fmt.Sprint("Allow{",allow.asstr(),"},Deny{",deny.asstr(),"}")
	
}


