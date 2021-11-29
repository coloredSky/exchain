package keeper

import (
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
	"github.com/okex/exchain/libs/cosmos-sdk/x/auth/exported"
)

// NewAccountWithAddress implements sdk.AccountKeeper.
func (ak AccountKeeper) NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) exported.Account {
	acc := ak.proto()
	err := acc.SetAddress(addr)
	if err != nil {
		panic(err)
	}
	return ak.NewAccount(ctx, acc)
}

// NewAccount sets the next account number to a given account interface
func (ak AccountKeeper) NewAccount(ctx sdk.Context, acc exported.Account) exported.Account {
	if err := acc.SetAccountNumber(ak.GetNextAccountNumber(ctx)); err != nil {
		panic(err)
	}
	return acc
}

// GetAccount implements sdk.AccountKeeper.
func (ak AccountKeeper) GetAccount(ctx sdk.Context, addr sdk.AccAddress) exported.Account {
	//store := ctx.KVStore(ak.key)
	//bz := store.Get(types.AddressStoreKey(addr))
	//if bz == nil {
	//	return nil
	//}
	//acc := ak.decodeAccount(bz)
	//return acc

	if ctx.IsCheckTx() {
		if val := ak.checkTxStore.Get(addr.String()); val != nil {
			return val
		}
	} else {
		if val := ak.deliverTxStore.Get(addr.String()); val != nil {
			return val
		}

		if val, ok := ak.accLRU.Get(addr.String()); ok {
			return val.(exported.Account)
		}
	}

	enc, err := ak.trie.TryGet(addr.Bytes())
	if err != nil {
		return nil
	}
	if len(enc) == 0 {
		return nil
	}

	var data []byte
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		return nil
	}

	return ak.decodeAccount(data)
}

// GetAllAccounts returns all accounts in the accountKeeper.
func (ak AccountKeeper) GetAllAccounts(ctx sdk.Context) (accounts []exported.Account) {
	ak.IterateAccounts(ctx,
		func(acc exported.Account) (stop bool) {
			accounts = append(accounts, acc)
			return false
		})
	return accounts
}

// SetAccount implements sdk.AccountKeeper.
func (ak AccountKeeper) SetAccount(ctx sdk.Context, acc exported.Account) {
	//addr := acc.GetAddress()
	//store := ctx.KVStore(ak.key)
	//bz, err := ak.cdc.MarshalBinaryBare(acc)
	//if err != nil {
	//	panic(err)
	//}
	//store.Set(types.AddressStoreKey(addr), bz)

	if ak.observers != nil && !ctx.IsCheckTx() {
		for _, observer := range ak.observers {
			if observer != nil {
				observer.OnAccountUpdated(acc)
			}
		}
	}

	if ctx.IsCheckTx() {
		ak.checkTxStore.Set(acc.GetAddress().String(), acc)
	} else {
		ak.deliverTxStore.Set(acc.GetAddress().String(), acc)
	}
}

// RemoveAccount removes an account for the account mapper store.
// NOTE: this will cause supply invariant violation if called
func (ak AccountKeeper) RemoveAccount(ctx sdk.Context, acc exported.Account) {
	//addr := acc.GetAddress()
	//store := ctx.KVStore(ak.key)
	//store.Delete(types.AddressStoreKey(addr))

	if ctx.IsCheckTx() {
		ak.checkTxStore.Delete(acc.GetAddress().String())
	} else {
		ak.deliverTxStore.Delete(acc.GetAddress().String())
	}
}

// IterateAccounts iterates over all the stored accounts and performs a callback function
func (ak AccountKeeper) IterateAccounts(ctx sdk.Context, cb func(account exported.Account) (stop bool)) {
	//store := ctx.KVStore(ak.key)
	//iterator := sdk.KVStorePrefixIterator(store, types.AddressStoreKeyPrefix)
	//
	//defer iterator.Close()
	//for ; iterator.Valid(); iterator.Next() {
	//	account := ak.decodeAccount(iterator.Value())
	//
	//	if cb(account) {
	//		break
	//	}
	//}

	it := trie.NewIterator(ak.trie.NodeIterator(nil))
	for it.Next() {
		if len(it.Value) > 0 {
			var data []byte
			if err := rlp.DecodeBytes(it.Value, &data); err != nil {
				continue
			}

			acc := ak.decodeAccount(data)
			if cb(acc) {
				break
			}
		}
	}
}

func (ak *AccountKeeper) Update(ctx sdk.Context, err error) {
	if !ctx.IsCheckTx() && err == nil {
		ak.deliverTxStore.IteratorCache(func(key string, acc exported.Account, isDirty bool, isDelete bool) {
			if !isDirty {
				return
			}

			accKey,_ := sdk.AccAddressFromBech32(key)
			if isDelete {
				ak.accLRU.Remove(key)

				// delete account
				ak.trie.TryDelete(accKey)

			} else {
				ak.accLRU.Add(key, acc)

				//update account
				value, err := ak.cdc.MarshalBinaryBare(acc)
				if err != nil {
					panic(err)
				}

				// Encode the account and update the account trie
				data, err := rlp.EncodeToBytes(value)
				if err != nil {
					panic(fmt.Errorf("can't encode object at %x: %v", key, err))
				}

				if err = ak.trie.TryUpdate(accKey, data); err != nil {
					panic(err)
				}
			}
		})
	}

	ak.CleanCacheStore()
}

func (ak *AccountKeeper) CleanCacheStore() {
	ak.checkTxStore.Clean()
	ak.deliverTxStore.Clean()
}
