// Copyright (c) 2017-2018 The nox developers
// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2015 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package base58

import (
	"errors"
	"reflect"
	"Nox-DAG-test/script/tool/hash"
)

// ErrChecksum indicates that the checksum of a check-encoded string does not verify against
// the checksum.
var ErrChecksum = errors.New("checksum error")

// ErrInvalidFormat indicates that the check-encoded string has an invalid format.
var ErrInvalidFormat = errors.New("invalid format: version and/or checksum bytes missing")

// checksum: first four bytes of double-BLAKEb.
func checksum_nox(input []byte) []byte {
	h := hash.DoubleHashB(input)
	var cksum [4]byte
	copy(cksum[:],h[:])
	return cksum[:]
}

func checksum_ss(input []byte) []byte {
	return SingleHashChecksumFunc(hash.GetHasher(hash.Blake2b_512),2)(input)
}

func SingleHashChecksumFunc(hasher hash.Hasher, cksum_size int) (func([]byte) []byte) {
	return func (input []byte) ([]byte) {
		h := hash.CalcHash(input,hasher)
		var cksum []byte
		cksum = append(cksum,h[:cksum_size]...)
		return cksum[:]
	}
}

func DoubleHashChecksumFunc(hasher hash.Hasher,cksum_size int) (func([]byte) []byte) {
	return func (input []byte) ([]byte) {
		first := hash.CalcHash(input,hasher)
		second := hash.CalcHash(first[:],hasher)
		var cksum []byte
		cksum = append(cksum,second[:cksum_size]...)
		return cksum[:]
	}
}

// CheckEncode prepends two version bytes and appends a four byte checksum.
func NoxCheckEncode(input []byte, version []byte) string {
	return CheckEncode(input,version[:],4,checksum_nox)
}

func CheckEncode(input []byte, version []byte, cksum_size int, cksumfunc func([]byte) []byte) string{
	b := make([]byte, 0, len(version)+len(input)+cksum_size)
	b = append(b, version[:]...)
	b = append(b, input[:]...)
	var cksum []byte
	cksum = cksumfunc(b)
	b = append(b, cksum[:]...)
	return Encode(b)
}

func CheckDecode(input string, version_size , cksum_size int, cksumfunc func([]byte) []byte) (result []byte, version []byte, err error) {
	decoded := Decode(input)
	if len(decoded) < cksum_size + version_size {
		return nil, []byte{}, ErrInvalidFormat
	}
	version = append(version,decoded[:version_size]...)
	var cksum []byte
	cksum = append(cksum, decoded[len(decoded)-cksum_size:]...)
	if !reflect.DeepEqual(cksumfunc(decoded[:len(decoded)-cksum_size]),cksum[:]) {
		return nil, []byte{}, ErrChecksum
	}
	payload := decoded[version_size : len(decoded)-cksum_size]
	result = append(result, payload...)
	return
}

// NoxCheckDecode decodes a string that was encoded with 2 bytes version and verifies
// the checksum using blake2b-256 hash.
func NoxCheckDecode(input string) (result []byte, version [2]byte, err error) {
	r,v,err := CheckDecode(input,2, 4,checksum_nox)
	if err!=nil{
		return nil, [2]byte{},err
	}
	return r, [2]byte{v[0],v[1]}, nil
}