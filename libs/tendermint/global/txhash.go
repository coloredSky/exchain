package global

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"sync"
)

type TxHashKeys struct {
	TxHash []byte
	Addr   ethcmn.Address
	Keys   []ethcmn.Hash
}

type PeerIdTxHash struct {
	txHashKeys sync.Map // peerId -> TxHashKeys
}
