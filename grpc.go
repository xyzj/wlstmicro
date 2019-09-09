package wlstmicro

import (
	"github.com/xyzj/gopsu/microgo"
	"google.golang.org/grpc"
)

// NewGRPCClient InitGRPCClient
// svraddr: 服务端地址
// cafiles: 依次为cert，key，clientca
func NewGRPCClient(svraddr string, cafiles ...string) (*grpc.ClientConn, error) {
	return microgo.NewGRPCClient(svraddr, cafiles...)
}

// NewGRPCServer InitGRPCServer
// cafiles: 依次为cert，key，clientca
func NewGRPCServer(cafiles ...string) (*grpc.Server, bool) {
	return microgo.NewGRPCServer(cafiles...)
}
