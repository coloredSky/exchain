package types

import (
	"fmt"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestEncodeToFile(t *testing.T) {
	for i := 0; i < 1000; i++ {
		addr := AddrKeys{
			Addr: ethcmn.Address{},
			Keys: []ethcmn.Hash{{0x01}, {0x02}, {0x03}, {0x04}},
		}
		StatisticsMap[fmt.Sprintf("%v", i)] = addr
	}
	//	err := EncodeToFile()
	//	t.Log(err)
}

func TestDecodeToMap(t *testing.T) {
	err := DecodeToMap()

	//	t.Log(err)
}
