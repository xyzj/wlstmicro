package wlstmicro

import (
	"github.com/xyzj/gopsu/microgo"
	"google.golang.org/grpc"
)

// InitGRPCClient InitGRPCClient
// svraddr: 服务端地址
// cafiles: 依次为cert，key，clientca
func InitGRPCClient(svraddr string, cafiles ...string) (*grpc.ClientConn, error) {
	return microgo.NewGRPCClient(svraddr, cafiles...)
}

// InitGRPCServer InitGRPCServer
// cafiles: 依次为cert，key，clientca
func InitGRPCServer(cafiles ...string) (*grpc.Server, bool) {
	return microgo.NewGRPCServer(cafiles...)
}
