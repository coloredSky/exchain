package types

import (
	"bytes"
	"encoding/gob"
	"errors"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"log"
	"os"
)

var (
	StatisticsMap    = make(map[ethcmn.Hash]map[ethcmn.Address][]ethcmn.Hash)
	FileName         = "./txhash"
	KeyTxCollectMode = false // true for collect tx
	ReplayStart      int64
	ReplayStop       int64
	Call             bool
	CurTxHash        ethcmn.Hash
)

func TotalAddr() int {
	totalAddr := 0
	for _, addr := range StatisticsMap {
		totalAddr += len(addr)
	}
	return totalAddr
}

func TotalKeys() int {
	totalKeys := 0
	for _, addr := range StatisticsMap {
		for _, keys := range addr {
			totalKeys += len(keys)
		}
	}

	return totalKeys
}

func Detail() {
	for _, addr := range StatisticsMap {
		perTxKeys := 0
		for _, keys := range addr {
			perTxKeys += len(keys)
		}
		log.Printf("tx: key count: %v \n", perTxKeys)
	}
}

func Print() {
	log.Printf("total tx %v,  addr %v, keys: %v", len(StatisticsMap), TotalAddr(), TotalKeys())
	Detail()
}

func AddMapTxHash(txHash ethcmn.Hash) {
	if _, ok := StatisticsMap[txHash]; !ok {
		StatisticsMap[txHash] = make(map[ethcmn.Address][]ethcmn.Hash)
	}
}

func AddMapAddrKey(addr ethcmn.Address, key ethcmn.Hash) {
	if _, ok := StatisticsMap[CurTxHash]; ok {
		StatisticsMap[CurTxHash][addr] = append(StatisticsMap[CurTxHash][addr], key)
	} else {
		log.Printf("not found txhash something went wrong...... %v %v %v\n", CurTxHash, addr, key)
	}
}

func EncodeToFile() error {
	f, err := os.OpenFile(FileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	b := new(bytes.Buffer)

	e := gob.NewEncoder(b)

	// Encoding the map
	err = e.Encode(StatisticsMap)
	if err != nil {
		return err
	}
	n, err := f.Write(b.Bytes())
	log.Printf("write to %v %v bytes", FileName, n)

	return err
}

func DecodeToMap() error {
	data, err := os.ReadFile(FileName)
	if err != nil {
		return err
	}
	d := gob.NewDecoder(bytes.NewBuffer(data))

	// Decoding the serialized data
	return d.Decode(&StatisticsMap)
}

func FileExist() bool {
	if _, err := os.Stat(FileName); err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		panic(err)

	}
	return false
}
