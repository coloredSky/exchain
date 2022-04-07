package types

import (
	"fmt"
	"github.com/okex/exchain/libs/cosmos-sdk/codec"
)

// Register the sdk message type
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterInterface((*Msg)(nil), nil)
	cdc.RegisterInterface((*MsgProtoAdapter)(nil), nil)
	cdc.RegisterInterface((*Tx)(nil), nil)
	cdc.RegisterConcrete(BaseTx{}, "cosmos-sdk/BaseTx", nil)
}

var (
	SLog = newScfLog()
)

type SCFLog struct {
	ll []string
}

func newScfLog() *SCFLog {
	return &SCFLog{
		ll: make([]string, 0),
	}
}

func (s *SCFLog) Add(data string) {
	s.ll = append(s.ll, data)
}

func (s *SCFLog) Clean() {
	s.ll = make([]string, 0)
}

func (s *SCFLog) Print() {
	fmt.Println("begin print log")
	for _, v := range s.ll {
		fmt.Println("v", v)
	}
}
