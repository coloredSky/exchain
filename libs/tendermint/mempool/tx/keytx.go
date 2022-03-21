package tx

type KeyTx struct {
	Payload []byte   `json:"payload"` // std tx or evm tx
	Keys    []string `json:"keys"`    // signature for payload
}

func (kx *KeyTx) GetPayload() []byte {
	if kx != nil {
		return kx.Payload
	}

	return nil
}

func (kx *KeyTx) GetKeys() []string {
	if kx != nil {
		return kx.Keys
	}

	return nil
}
