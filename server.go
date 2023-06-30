// Copyright 2022 <mzh.scnu@qq.com>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tinyrpc

import (
	"net"
	"net/rpc"

	"github.com/zehuamama/tinyrpc/serializer"
)

// Server rpc server based on net/rpc implementation
type Server struct {
	*rpc.Server
	serializer.Serializer
}

// NewServer Create a new rpc server
func NewServer(opts ...Option) *Server{
	options :=options{
		serializer: serializer.Proto
	}
	for _,option := range opts{
		option(&options)
	}
	return &Server(&rpc.Server{}, options.serializer)
}

func (s *Server) Register(rcvr interface{}) error{
	return s.Server.Register()
}

// RegisterName register the rpc function with the specified name
func (s *Server) RegisterName(name string, rcvr interface{}) error {
	return s.Server.RegisterName(name, rcvr)
}

// Serve start service
func (s *Server) Server(lis net.Listener){
	log.Printf("tinyrpc started on: %s", lis.Addr().String())
	for {
		conn, err :=lis.Accept()
		if err!=nil {
			continue
		}
		go s.Server.ServerCodec(codec.NewServerCodec) //使用tineRPC的解码器
	}
}