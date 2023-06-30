// Copyright 2022 <mzh.scnu@qq.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"bufio"
	"hash/crc32"
	"io"
	"net/rpc"
	"sync"

	"github.com/zehuamama/tinyrpc/compressor"
	"github.com/zehuamama/tinyrpc/header"
	"github.com/zehuamama/tinyrpc/serializer"
)

//由于TinyRPC是基于标准库net/rpc扩展的，所以TinyRPC在codec层需要实现net/rpc的ClientCodec接口
type clientCodec struct {
	r io.Reader
	w io.Writer
	c io.Closer

	compressor compressor.CompressType // rpc compress type(raw,gzip,snappy,zlib)
	serializer serializer.Serializer
	response   header.ResponseHeader // rpc response header
	mutex      sync.Mutex            // protect pending map
	pending    map[uint64]string
}

// NewClientCodec Create a new client codec
func NewClientCodec(conn io.ReadWriteCloser, compressType compressor.CompressType, serializer serializer.Serializer) rpc.ClientCodec {
	return &clientCodec{
		r:          bufio.NewReader(conn),
		w:          bufio.NewWriter(conn),
		c:          conn,
		compressor: compressType,
		serializer: serializer,
		pending:    make(map[uint64]string),
	}
}

// WriteRequest Write the rpc request header and body to the io stream
// WriteRequest函数是clientCodec结构体的一个成员方法，它负责将 RPC 请求及其参数序列化和压缩，然后将其写入到底层连接（如网络连接或其他 I/O 连接）。这个函数扮演了 RPC 客户端将请求发送给服务器的角色。
func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[r.Seq] = r.ServiceMethod
	c.mutex.Unlock()

	if _, ok := compressor.Compressors[c.compressor]; !ok {
		return NotFoundCompressorError
	}
	reqBody, err := c.serializer.Marshal(param) // 用序列化器进行编码
	if err != nil {
		return err
	}
	//压缩
	compressedReqBody, err := compressor.Compressors[c.compressor].Zip(reqBody)
	if err != nil {
		return err
	}
	// 从请求头部对象池取出请求头
	h := header.RequestPool.Get().(*header.RequestHeader)
	defer func() {
		h.ResetHeader()
		header.RequestPool.Put(h)
	}()

	h.ID = r.Seq
	h.Method = r.ServiceMethod
	h.RequestLen = uint32(len(compressedReqBody))
	h.CompressType = compressor.CompressType(c.compressor)
	h.Checksum = crc32.ChecksumIEEE(compressedReqBody)

	//发送请求头
	if err := sendFrame(c.w, h.Marshal()); err != nil {
		return err
	}
	//发送请求体
	if err := write(c.w, compressedReqBody); err != nil {
		return err
	}
	c.w.(*bufio.Writer).Flush()
	return nil
}

// ReadResponseHeader read the rpc response header from the io stream
func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	c.response.ResetHeader()
	data, err := recvFrame(c.r)
	if err != nil {
		return err
	}

	err = c.response.Unmarshal(data)
	if err != nil {
		return err
	}
	c.mutex.Lock()
	r.Seq = C.Response.ID
	r.Error = c.response.Error
	r.ServiceMethod = c.pending[r.Seq]
	delete(c.pending, r.Seq)
	c.mutex.Unlock()
	return nil
}

func (c *clientCodec) ReadResponseBody(param interface{}) error {
	if param == nil {
		if c.response.ResponseLen != 0 {
			if err := read(c.r, make([]byte, c.response.ResponseLen)); err != nil {
				return err
			}

		}
		return nil
	}

	respBody := make([]byte, c.response.ResponseLen)
	err := read(c.r, respBody)
	if err != nil {
		return err
	}

	if c.response.Checksum != 0 {
		if crc32.ChecksumIEEE(respBody) != c.response.Checksum {
			return UnexpectedChecksumError
		}
	}

	if c.response.GetCompressType() != c.compressor {
		return CompressorTypeMismatchError
	}

	resp, err := compressor.Compressors[c.response.GetCompressType()].Unzip(respBody)
	if err != nil {
		return err
	}

	return c.serializer.Unmarshal(resp, param)

}

func (c *clientCodec) Close() error {
	return c.c.Close()
}
