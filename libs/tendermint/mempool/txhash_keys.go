package mempool

import "sync"

type TxHashKeys struct {
	txHashKeys sync.Map // txHash -> tx.KeysTx
}
