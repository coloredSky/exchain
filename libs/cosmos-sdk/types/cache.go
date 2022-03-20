package types

import (
	"bytes"
	"encoding/hex"
	"fmt"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/okex/exchain/libs/cosmos-sdk/store/types"
	"github.com/okex/exchain/libs/tendermint/crypto"
	"github.com/spf13/viper"
	"sync"
	"time"
)

var (
	maxAccInMap        = 100000
	deleteAccCount     = 10000
	maxStorageInMap    = 10000000
	deleteStorageCount = 1000000

	FlagMultiCache         = "multi-cache"
	MaxAccInMultiCache     = "multi-cache-acc"
	MaxStorageInMultiCache = "multi-cache-storage"
	UseCache               bool
)

type account interface {
	Copy() interface{}
	GetAddress() AccAddress
	SetAddress(AccAddress) error
	GetPubKey() crypto.PubKey
	SetPubKey(crypto.PubKey) error
	GetAccountNumber() uint64
	SetAccountNumber(uint64) error
	GetSequence() uint64
	SetSequence(uint64) error
	GetCoins() Coins
	SetCoins(Coins) error
	SpendableCoins(blockTime time.Time) Coins
	String() string
}

type storageWithCache struct {
	Value  []byte
	Dirty  bool
	Delete bool
}

type accountWithCache struct {
	Acc      account
	Gas      uint64
	Bz       []byte
	IsDirty  bool
	ISDelete bool
}

type codeWithCache struct {
	Code    []byte
	IsDirty bool
}

type Cache struct {
	mu sync.Mutex

	useCache  bool
	parent    *Cache
	gasConfig types.GasConfig

	dirtyStorageMap map[ethcmn.Address]map[ethcmn.Hash]*storageWithCache
	readStorageMap  map[ethcmn.Address]map[ethcmn.Hash][]byte

	dirtyaccMap map[ethcmn.Address]*accountWithCache
	readaccMap  map[ethcmn.Address]*accountWithCache

	dirtycodeMap map[ethcmn.Hash]*codeWithCache
	readcodeMap  map[ethcmn.Hash][]byte
}

func initCacheParam() {
	UseCache = viper.GetBool(FlagMultiCache)

	if data := viper.GetInt(MaxAccInMultiCache); data != 0 {
		maxAccInMap = data
		deleteAccCount = maxAccInMap / 10
	}

	if data := viper.GetInt(MaxStorageInMultiCache); data != 0 {
		maxStorageInMap = data
		deleteStorageCount = maxStorageInMap / 10
	}
}

func NewChainCache() *Cache {
	initCacheParam()
	return NewCache(nil, true)
}

func NewCache(parent *Cache, useCache bool) *Cache {
	return &Cache{
		mu: sync.Mutex{},

		useCache: useCache,
		parent:   parent,

		dirtyStorageMap: make(map[ethcmn.Address]map[ethcmn.Hash]*storageWithCache, 0),
		readStorageMap:  make(map[ethcmn.Address]map[ethcmn.Hash][]byte, 0),

		dirtyaccMap: make(map[ethcmn.Address]*accountWithCache, 0),
		readaccMap:  make(map[ethcmn.Address]*accountWithCache, 0),

		dirtycodeMap: make(map[ethcmn.Hash]*codeWithCache),
		readcodeMap:  make(map[ethcmn.Hash][]byte),
		gasConfig:    types.KVGasConfig(),
	}

}

func (c *Cache) UseCache() bool {
	return !c.skip()
}

func (c *Cache) skip() bool {
	if c == nil || !c.useCache {
		return true
	}
	return false
}

func (c *Cache) UpdateAccount(addr AccAddress, acc account, bz []byte, isDirty bool, isDelete bool) {
	if c.skip() {
		return
	}
	ethAddr := ethcmn.BytesToAddress(addr.Bytes())

	tt := &accountWithCache{
		Acc:      acc,
		IsDirty:  isDirty,
		ISDelete: isDelete,
		Bz:       bz,
		Gas:      types.Gas(len(bz))*c.gasConfig.ReadCostPerByte + c.gasConfig.ReadCostFlat,
	}

	c.mu.Lock()
	if !isDirty {
		c.setReadAccount(ethAddr, acc, bz, tt.Gas)
	} else {
		c.dirtyaccMap[ethAddr] = tt
	}
	c.mu.Unlock()
}

func (c *Cache) UpdateStorage(addr ethcmn.Address, key ethcmn.Hash, value []byte, isDirty bool, isDelete bool) {
	//fmt.Println("uuuuuu", addr.String(), key.String(), isDirty, isDelete)
	if c.skip() {
		//fmt.Println("skip----")
		return
	}

	c.mu.Lock()
	if isDirty {
		if _, ok := c.dirtyStorageMap[addr]; !ok {
			c.dirtyStorageMap[addr] = make(map[ethcmn.Hash]*storageWithCache, 0)
		}

		c.dirtyStorageMap[addr][key] = &storageWithCache{
			Value:  value,
			Dirty:  isDirty,
			Delete: isDelete,
		}
	} else {
		if addr.String() == "0xadf4916d11F352a2748e19F3056428639313F6E1" {
			//fmt.Println("fuckhere")
		}
		c.setReadStorage(addr, key, value)
	}
	c.mu.Unlock()
}

func (c *Cache) UpdateCode(key []byte, value []byte, isdirty bool) {
	if c.skip() {
		return
	}
	hash := ethcmn.BytesToHash(key)
	c.mu.Lock()
	if isdirty {
		c.dirtycodeMap[hash] = &codeWithCache{
			Code:    value,
			IsDirty: isdirty,
		}
	} else {
		c.SetReadCode(hash, value)
	}

	c.mu.Unlock()
}

func (c *Cache) GetAccount(addr ethcmn.Address) (account, uint64, []byte, bool) {
	if c.skip() {
		//fmt.Println("ski[-")
		return nil, 0, nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if data, ok := c.dirtyaccMap[addr]; ok {
		//fmt.Println("????-1", addr.String(), ok)
		return data.Acc, data.Gas, data.Bz, ok
	}

	if data, ok := c.readaccMap[addr]; ok {
		//fmt.Println("????-2", addr.String(), data.Acc == nil, data.Gas, len(data.Bz), ok)
		return data.Acc, data.Gas, data.Bz, ok
	}

	if c.parent != nil {
		acc, gas, bz, ok := c.parent.GetAccount(addr)
		if ok {
			c.setReadAccount(addr, acc, bz, gas)
		}

		return acc, gas, bz, ok
	}

	//fmt.Println("228-0-0----")
	return nil, 0, nil, false
}

func (c *Cache) setReadAccount(addr ethcmn.Address, acc account, bz []byte, gas uint64) {
	if addr.String() == "0xf1829676DB577682E944fc3493d451B67Ff3E29F" {
		//fmt.Println("set read account---", addr.String(), acc == nil, len(bz), gas)
		//debug.PrintStack()
	}
	c.readaccMap[addr] = &accountWithCache{
		Acc:     acc,
		Gas:     gas,
		Bz:      bz,
		IsDirty: false,
	}
}

func (c *Cache) setReadStorage(addr ethcmn.Address, key ethcmn.Hash, value []byte) {
	if addr.String() == "0xadf4916d11F352a2748e19F3056428639313F6E1" {
		//fmt.Println("?????--set read Storage", addr.String(), key.String(), hex.EncodeToString(value))
	}
	if _, ok := c.readStorageMap[addr]; !ok {
		c.readStorageMap[addr] = make(map[ethcmn.Hash][]byte)
	}
	c.readStorageMap[addr][key] = value
}

func (c *Cache) SetReadCode(hash ethcmn.Hash, value []byte) {
	c.readcodeMap[hash] = value
}
func (c *Cache) GetStorage(addr ethcmn.Address, key ethcmn.Hash) ([]byte, bool) {
	if c.skip() {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	pringLog := addr.String() == "0xadf4916d11F352a2748e19F3056428639313F6E1"
	if pringLog {
		//fmt.Println("GetStorage", addr.String(), key.String(), c.parent == nil)
	}

	if _, hasAddr := c.dirtyStorageMap[addr]; hasAddr {
		data, hasKey := c.dirtyStorageMap[addr][key]
		if hasKey {
			if pringLog {
				//fmt.Println("result-1")
			}

			return data.Value, hasKey
		}
	} else {
		c.dirtyStorageMap[addr] = make(map[ethcmn.Hash]*storageWithCache)
	}

	if _, hasAddr := c.readStorageMap[addr]; hasAddr {
		if data, hasKey := c.readStorageMap[addr][key]; hasKey {
			if pringLog {
				//fmt.Println("result-2")
			}

			return data, true
		}
	}

	if c.parent != nil {
		value, ok := c.parent.GetStorage(addr, key)
		if ok {
			if pringLog {
				//fmt.Println("here---", key.String(), hex.EncodeToString(value))
			}
			c.setReadStorage(addr, key, value)
		}
		if pringLog {
			//fmt.Println("result-3")
		}
		return value, ok
	}
	if addr.String() == "0xadf4916d11F352a2748e19F3056428639313F6E1" {
		//fmt.Println("no fund", addr.String(), key.String(), c.parent == nil)
	}

	if pringLog {
		//fmt.Println("result-4")
	}
	return nil, false
}

func (c *Cache) GetCode(key []byte) ([]byte, bool) {
	if c.skip() {
		return nil, false
	}

	hash := ethcmn.BytesToHash(key)
	c.mu.Lock()
	defer c.mu.Unlock()
	if data, ok := c.dirtycodeMap[hash]; ok {
		return data.Code, ok
	}

	if data, ok := c.readcodeMap[hash]; ok {
		return data, ok
	}
	if c.parent != nil {
		code, ok := c.parent.GetCode(hash.Bytes())
		if ok {
			c.SetReadCode(hash, code)
		}
		return code, ok
	}
	return nil, false
}

func (c *Cache) GetDirtyAcc() map[ethcmn.Address]*accountWithCache {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dirtyaccMap
}

func (c *Cache) GetDirtyCode() map[ethcmn.Hash]*codeWithCache {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dirtycodeMap
}

func (c *Cache) GetDirtyStorage() map[ethcmn.Address]map[ethcmn.Hash]*storageWithCache {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dirtyStorageMap
}
func (c *Cache) Write(updateDirty bool, printLog bool) {
	if c.skip() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.parent == nil {
		return
	}

	c.writeStorage(c.parent, updateDirty, printLog)
	c.writeAcc(c.parent, updateDirty)
	c.writeCode(c.parent, updateDirty)
}

type ReadList struct {
	Account map[ethcmn.Address][]byte
	Storage map[ethcmn.Address]map[ethcmn.Hash][]byte
	Code    map[ethcmn.Hash][]byte
}

func (c *Cache) CopyRead() *ReadList {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := &ReadList{
		Account: make(map[ethcmn.Address][]byte),
		Storage: make(map[ethcmn.Address]map[ethcmn.Hash][]byte),
		Code:    make(map[ethcmn.Hash][]byte),
	}
	for addr, v := range c.readaccMap {
		s.Account[addr] = v.Bz
	}
	for addr, v := range c.readStorageMap {
		s.Storage[addr] = make(map[ethcmn.Hash][]byte, 0)
		for kk, vv := range v {
			s.Storage[addr][kk] = vv
		}
	}
	for hash, code := range c.readcodeMap {
		s.Code[hash] = code
	}
	return s
}

func (c *Cache) WriteToNewCache(newCache *Cache) {
	if c.skip() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	newCache.mu.Lock()
	defer newCache.mu.Unlock()

	c.writeStorage(newCache, true, true)
	c.writeAcc(newCache, true)
	c.writeCode(newCache, true)

}

func (c *Cache) writeStorage(parent *Cache, updateDirty bool, printLog bool) {
	for addr, storages := range c.dirtyStorageMap {
		if _, ok := parent.dirtyStorageMap[addr]; !ok {
			parent.dirtyStorageMap[addr] = make(map[ethcmn.Hash]*storageWithCache, 0)
		}

		for key, v := range storages {
			if updateDirty {
				if printLog {
					//if addr.String() == "0xadf4916d11F352a2748e19F3056428639313F6E1" {
					//	fmt.Println("write -addr", addr.String(), key.String(), hex.EncodeToString(v.Value))
					//}
				}
				parent.dirtyStorageMap[addr][key] = v
			}
		}
	}

	for addr, storages := range c.readStorageMap {
		if _, ok := parent.readStorageMap[addr]; !ok {
			parent.readStorageMap[addr] = make(map[ethcmn.Hash][]byte, 0)
		}

		for key, v := range storages {
			if updateDirty {
				if _, ok := parent.readStorageMap[addr][key]; !ok {
					parent.readStorageMap[addr][key] = v
				}
			}
		}
	}

	c.dirtyStorageMap = make(map[ethcmn.Address]map[ethcmn.Hash]*storageWithCache)
	c.readStorageMap = make(map[ethcmn.Address]map[ethcmn.Hash][]byte)
}

func (c *Cache) writeAcc(parent *Cache, updateDirty bool) {
	for addr, v := range c.dirtyaccMap {
		if updateDirty {
			parent.dirtyaccMap[addr] = v
		}
	}

	for addr, v := range c.readaccMap {
		if updateDirty {
			if _, ok := parent.readaccMap[addr]; !ok {
				//fmt.Println("writeAcc", addr.String(), v.Acc == nil)
				parent.readaccMap[addr] = v
			}
		}
	}
	c.dirtyaccMap = make(map[ethcmn.Address]*accountWithCache)
	c.readaccMap = make(map[ethcmn.Address]*accountWithCache)
}

func (c *Cache) writeCode(parent *Cache, updateDirty bool) {
	for hash, v := range c.dirtycodeMap {
		if updateDirty {
			parent.dirtycodeMap[hash] = v
		}
	}
	for hash, v := range c.readcodeMap {
		if updateDirty {
			if _, ok := parent.readcodeMap[hash]; !ok {
				parent.readcodeMap[hash] = v
			}
		}
	}
	c.dirtycodeMap = make(map[ethcmn.Hash]*codeWithCache)
	c.readcodeMap = make(map[ethcmn.Hash][]byte)
}

func (c *Cache) IsConflict(newCache *ReadList, whiteAddr ethcmn.Address) bool {
	//fmt.Println("readStorageMap", len(newCache.readaccMap), len(newCache.readStorageMap), len(newCache.readcodeMap))

	c.mu.Lock()
	defer c.mu.Unlock()

	for acc, v := range newCache.Account {
		if acc == whiteAddr {
			continue
		}
		if data, ok := c.dirtyaccMap[acc]; ok && data.IsDirty {
			if !bytes.Equal(v, data.Bz) {
				fmt.Println("conflict-acc", acc.String())
				return true
			}
		}
	}

	for acc, ss := range newCache.Storage {
		//fmt.Println("readStorageMap", acc)
		preSS, ok := c.dirtyStorageMap[acc]
		if !ok {
			continue
		}
		for kk, vv := range ss {
			if kk.String() == "0x21feabc92835ec5eac6c73ca8350bf06d78c2088a72988aaa1e56fede3a81fb3" {
				//fmt.Println("read---", kk.String(), hex.EncodeToString(vv))
			}

			if pp, ok1 := preSS[kk]; ok1 && pp.Dirty {
				if !bytes.Equal(pp.Value, vv) {
					fmt.Println("conflict-storage", acc.String(), kk.String(), "now", hex.EncodeToString(pp.Value), "read", hex.EncodeToString(vv))
					return true
				}
			}

		}
	}

	for acc, code := range newCache.Code {
		if data, ok := c.dirtycodeMap[acc]; ok && data.IsDirty {
			if !bytes.Equal(code, data.Code) {
				fmt.Println("conflict-code", acc.String())
				return true
			}
		}
	}
	return false
}

//func (c *Cache) TryDelete(logger log.Logger, height int64) {
//	if c.skip() {
//		return
//	}
//	if height%100 == 0 {
//		c.logInfo(logger, "null")
//	}
//
//	lenStorage := c.storageSize()
//	if len(c.accMap) < maxAccInMap && lenStorage < maxStorageInMap {
//		return
//	}
//
//	deleteMsg := ""
//	if len(c.accMap) >= maxAccInMap {
//		deleteMsg += fmt.Sprintf("Acc:Deleted Before:%d", len(c.accMap))
//		cnt := 0
//		for key := range c.accMap {
//			delete(c.accMap, key)
//			cnt++
//			if cnt > deleteAccCount {
//				break
//			}
//		}
//	}
//
//	if lenStorage >= maxStorageInMap {
//		deleteMsg += fmt.Sprintf("Storage:Deleted Before:len(contract):%d, len(storage):%d", len(c.storageMap), lenStorage)
//		cnt := 0
//		for key, value := range c.storageMap {
//			cnt += len(value)
//			delete(c.storageMap, key)
//			if cnt > deleteStorageCount {
//				break
//			}
//		}
//	}
//	if deleteMsg != "" {
//		c.logInfo(logger, deleteMsg)
//	}
//}

//func (c *Cache) logInfo(logger log.Logger, deleteMsg string) {
//	nowStats := fmt.Sprintf("len(acc):%d len(contracts):%d len(storage):%d", len(c.accMap), len(c.storageMap), c.storageSize())
//	logger.Info("MultiCache", "deleteMsg", deleteMsg, "nowStats", nowStats)
//}

func (c *Cache) GetParent() *Cache {
	return c.parent
}

func (c *Cache) Print(printLog bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//fmt.Println("size::::", len(c.dirtyaccMap), len(c.dirtyStorageMap), len(c.dirtycodeMap))
	//fmt.Println("size::::", len(c.readaccMap), len(c.readStorageMap), len(c.readcodeMap))

	if !printLog {
		return
	}
	for acc, v := range c.dirtyaccMap {
		fmt.Println("acc:", acc.String(), v.IsDirty)
	}
	for acc, v := range c.dirtyStorageMap {
		for kk, vv := range v {
			fmt.Println("storage:", acc.String(), kk.String(), hex.EncodeToString(vv.Value), vv.Dirty)
		}
	}
	for acc, _ := range c.dirtycodeMap {
		fmt.Println("code:", acc.String())
	}
	//for acc, v := range c.accMap {
	//	fmt.Println("acc", acc.String(), v.isDirty)
	//}
	//
	//for acc, v := range c.storageMap {
	//	fmt.Println("storage", acc.String(), v)
	//}
	//for acc, v := range c.codeMap {
	//	fmt.Println("code", acc.String(), v.isDirty)
	//}
}
