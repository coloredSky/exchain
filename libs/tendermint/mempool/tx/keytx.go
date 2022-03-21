package tx

import ethcmn "github.com/ethereum/go-ethereum/common"

type KeyTx struct {
	Payload []byte        `json:"payload"` // std tx or evm tx
	Keys    []ethcmn.Hash `json:"keys"`    // signature for payload
}

func (kx *KeyTx) GetPayload() []byte {
	if kx != nil {
		return kx.Payload
	}

	return nil
}

func (kx *KeyTx) GetKeys() []ethcmn.Hash {
	if kx != nil {
		return kx.Keys
	}

	return nil
}
