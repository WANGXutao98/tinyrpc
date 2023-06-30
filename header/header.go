// Copyright 2022 <mzh.scnu@qq.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package header

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/zehuamama/tinyrpc/compressor"
)

const (
	// MaxHeaderSize = 2 + 10 + 10 + 10 + 4 (10 refer to binary.MaxVarintLen64)
	MaxHeaderSize = 36

	Uint32Size = 4
	Uint16Size = 2
)

var UnmarshalError = errors.New("an error occurred in Unmarshal")

// RequestHeader request header structure looks like:
// +--------------+----------------+----------+------------+----------+
// | CompressType |      Method    |    ID    | RequestLen | Checksum |
// +--------------+----------------+----------+------------+----------+
// |    uint16    | uvarint+string |  uvarint |   uvarint  |  uint32  |
// +--------------+----------------+----------+------------+----------+
type RequestHeader struct {
	sync.RWMutex
	CompressType compressor.CompressType
	Method       string
	ID           uint64
	RequestLen   uint32
	Checksum     uint32
}

// Marshal will encode request header into a byte slice
func (r *RequestHeader) Marshal() []byte {
	//是 Go 语言中 sync.RWMutex 类型的一个方法，它用于在读取共享资源时获取读锁。RWMutex 是一个读/写互斥锁，它允许多个读操作并行进行，但写操作是独占的。
	r.RLock()
	defer r.RUnlock()
	idx := 0
	header := make([]byte, MaxHeaderSize+len(r.Method))

	binary.LittleEndian.PutUint16(header[idx:], uint16(r.CompressType))
	idx += Uint16Size
	idx += writeString(header[idx:], r.Method)
	idx += binary.PutUvarint(header[idx:], r.ID)
	idx += binary.PutUvarint(header[idx:], uint64(r.RequestLen))

	binary.LittleEndian.PutUint32(header[idx:], r.Checksum)
	idx += Uint32Size
	return header[:idx]

}

func (r *RequestHeader) UnMarshal(data []byte) (err error) {
	r.Lock()
	defer r.Unlock()
	if len(data) == 0 {
		return UnmarshalError
	}

	//捕获panic
	defer func() {
		if r := recover(); r != nil {
			err = UnmarshalError
		}
	}()

	idx, size := 0, 0
	r.CompressType = compressor.CompressType(binary.LittleEndian.Uint16(data[idx:]))
	idx += Uint16Size

	r.ID, size = binary.Uvarint(data[idx:])
	idx += size

	length, size := binary.Uvarint(data[idx:])
	r.RequestLen = uint32(length)
	idx += size

	r.Checksum = binary.LittleEndian.Uint32(data[idx:])
	return

}

func (r *RequestHeader) GetCompressType() compressor.CompressType {
	r.RLock()
	defer r.Unlock()
	return compressor.CompressType(r.CompressType)
}

// ResetHeader reset request header
func (r *RequestHeader) ResetHeader() {
	r.Lock()
	defer r.Unlock()
	r.ID = 0
	r.Checksum = 0
	r.Method = ""
	r.CompressType = 0
	r.RequestLen = 0
}

// ResponseHeader request header structure looks like:
// +--------------+---------+----------------+-------------+----------+
// | CompressType |    ID   |      Error     | ResponseLen | Checksum |
// +--------------+---------+----------------+-------------+----------+
// |    uint16    | uvarint | uvarint+string |    uvarint  |  uint32  |
// +--------------+---------+----------------+-------------+----------+
type ResponseHeader struct {
	sync.RWMutex
	CompressType compressor.CompressType
	ID           uint64
	Error        string
	ResponseLen  uint32
	Checksum     uint32
}

func (r *ResponseHeader) Unmarshal(data []byte) (err error) {
	r.Lock()
	defer r.Unlock()
	if len(data) == 0 {
		return UnmarshalError
	}
	defer func() {
		if r := recover(); r != nil {
			err = UnmarshalError
		}

	}()
	idx, size := 0, 0
	r.CompressType = compressor.CompressType(binary.LittleEndian.Uint16(data[idx:]))
	idx += Uint16Size

	r.ID, size = binary.Uvarint(data[idx:])
	idx += size

	r.Error, size = readString(data[idx:])
	idx += size

	length, size := binary.Uvarint(data[idx:])
	r.ResponseLen = uint32(length)
	idx += size

	r.Checksum = binary.LittleEndian.Uint32(data[idx:])
	return

}

func (r *ResponseHeader) GetCompressType() compressor.CompressType {
	r.RLock()
	defer r.Unlock()
	return compressor.CompressType(r.CompressType)
}

// ResetHeader reset response header
func (r *ResponseHeader) ResetHeader() {
	r.Lock()
	defer r.Unlock()
	r.Error = ""
	r.ID = 0
	r.CompressType = 0
	r.Checksum = 0
	r.ResponseLen = 0
}

func readString(data []byte) (string, int) {
	idx := 0
	length, size := binary.Uvarint(data)
	idx += size
	str := string(data[idx : idx+int(length)])
	idx += len(str)
	return str, idx
}

func writeString(data []byte, str string) int {
	idx := 0
	idx += binary.PutUvarint(data, uint64(len(str)))

	copy(data[idx:], str)
	idx += len(str)
	return idx
}

/*
func PutUvarint(buf []byte, x uint64) int

参数列表：
1）buf  需写入的缓冲区
2）x  uint64类型数字
返回值：
1）int  写入字节数。
2）panic  buf过小。
功能说明：
PutUvarint主要是讲uint64类型放入buf中，并返回写入的字节数。如果buf过小，PutUvarint将抛出panic。

*/
