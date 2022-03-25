package tx

import (
	"github.com/ethereum/go-ethereum/common"
)

type KtxMessage struct {
	*KeysTx
}

type KeyTxMeta struct {
	EthAddr common.Address `json:"eth_addr"`
	Keys    []common.Hash  `json:"keys"` // keys the original tx will read
}

type KeysTx struct {
	// todo the original tx could be wtx ?
	OriginalTx []byte      `json:"original_tx"` // std tx or evm tx
	AddrKeys   []KeyTxMeta `json:"addr_keys"`
}

func (kx *KeysTx) GetOriginalTx() []byte {
	if kx != nil {
		return kx.OriginalTx
	}

	return nil
}
