// Copyright 2022 <mzh.scnu@qq.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package header

import "sync"

var (
	RequestPool  sync.Pool
	ResponsePool sync.Pool
)

func init() {
	RequestPool = sync.Pool{New: func() interface{} {
		return &RequestHeader
	}}
	ResponsePool = sync.Pool{New: func() interface{} {
		return &ResponseHeader
	}}

}

/*
为了减少创建请求头部对象 RequestHeader 和响应头部对象 ResponseHeader 的次数，我们通过为这两个结构体建立对象池，以便可以进行复用。

同时我们为 RequestHeader 和 ResponseHeader 都实现了ResetHeader方法，当每次使用完这些对象时，我们调用ResetHeader让结构体内容初始化，随后再把它们丢回对象池里。
*/
