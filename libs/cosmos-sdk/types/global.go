package types

import (
	"bytes"
	"encoding/gob"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"log"
	"os"
)

type AddrKeys struct {
	Addr ethcmn.Address
	Keys []ethcmn.Hash
}

var (
	StatisticsMap = make(map[string]AddrKeys)
	FileName      = "./txhash"
)

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
