package types

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestEncodeToFile(t *testing.T) {
	for i := 0; i < 1; i++ {
		StatisticsMap[ethcmn.Hash{byte(i)}] = make(map[ethcmn.Address][]ethcmn.Hash)
		StatisticsMap[ethcmn.Hash{byte(i)}][ethcmn.Address{}] = []ethcmn.Hash{{0x01}, {0x02}, {0x03}, {0x04}}
		StatisticsMap[ethcmn.Hash{byte(i)}][ethcmn.Address{0x02}] = []ethcmn.Hash{{0x01}, {0x02}, {0x03}, {0x04}}
	}
	err := EncodeToFile()
	t.Log(err)
}

func TestDecodeToMap(t *testing.T) {
	err := DecodeToMap()

	t.Log(err)
}

func TestFileExist(t *testing.T) {
	t.Log(FileExist())
}

func TestAddMapAddrKey(t *testing.T) {
	AddMapTxHash(ethcmn.Hash{0x01})
	CurTxHash = ethcmn.Hash{0x01}
	AddMapAddrKey(ethcmn.Address{0x02}, ethcmn.Hash{0x03})
	AddMapAddrKey(ethcmn.Address{0x02}, ethcmn.Hash{0x04})
	AddMapAddrKey(ethcmn.Address{0x03}, ethcmn.Hash{0x05})
	AddMapAddrKey(ethcmn.Address{0x03}, ethcmn.Hash{0x06})
	t.Log(StatisticsMap)
}
