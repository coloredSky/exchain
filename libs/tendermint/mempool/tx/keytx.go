package tx

import "github.com/ethereum/go-ethereum/common"

type KtxMessage struct {
	*KeysTx
}

type KeysTx struct {
	// todo the original tx could be wtx
	OriginalTx []byte        `json:"original_tx"` // std tx or evm tx
	Keys       []common.Hash `json:"keys"`        // keys the original tx will read
}

func (kx *KeysTx) GetOriginalTx() []byte {
	if kx != nil {
		return kx.OriginalTx
	}

	return nil
}

func (kx *KeysTx) GetKeys() []common.Hash {
	if kx != nil {
		return kx.Keys
	}

	return nil
}
