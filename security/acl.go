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

type AccessControlList map[SID]AccessControlVector

type AccessControlEntry struct{
	Subject SID
	Rights  AccessControlVector
}

func (a AccessControlList) GetRights(s []SID) AccessControlVector {
	v := AccessControlVector(0)
	for _,i := range s {
		v |= a[i]
	}
	return v
}
func (a AccessControlList) GetRightsTypeEnforcement(aces []AccessControlEntry) AccessControlVector {
	v := AccessControlVector(0)
	for _,ace := range aces {
		v |= (a[ace.Subject] & ace.Rights)
	}
	return v
}
func (a AccessControlList) AddEntry(ace AccessControlEntry) {
	a[ace.Subject] = (a[ace.Subject] | ace.Rights)
}
func (a AccessControlList) AddEntries(aces []AccessControlEntry) {
	for _,ace := range aces {
		a[ace.Subject] = (a[ace.Subject] | ace.Rights)
	}
}
func (a AccessControlList) GetEntries() (aces []AccessControlEntry) {
	aces = make([]AccessControlEntry,0,len(a))
	for sid,acv := range a {
		if acv==AccessControlVector(0) { continue }
		aces = append(aces,AccessControlEntry{sid,acv})
	}
	return
}



