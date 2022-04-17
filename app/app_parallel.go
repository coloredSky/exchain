package app

import (
	ethcmn "github.com/ethereum/go-ethereum/common"
	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
	"github.com/okex/exchain/libs/cosmos-sdk/x/auth"
	authante "github.com/okex/exchain/libs/cosmos-sdk/x/auth/ante"
	"github.com/okex/exchain/libs/cosmos-sdk/x/bank"
	"github.com/okex/exchain/libs/cosmos-sdk/x/supply"
	"github.com/okex/exchain/x/evm"
	evmtypes "github.com/okex/exchain/x/evm/types"
)

// feeCollectorHandler set or get the value of feeCollectorAcc
func updateFeeCollectorHandler(bk bank.Keeper, sk supply.Keeper) sdk.UpdateFeeCollectorAccHandler {
	return func(ctx sdk.Context, balance sdk.Coins) error {
		return bk.SetCoins(ctx, sk.GetModuleAddress(auth.FeeCollectorName), balance)
	}
}

// evmTxFeeHandler get tx fee for evm tx
func evmTxFeeHandler(ak auth.AccountKeeper, ek *evm.Keeper) sdk.GetTxFeeHandler {
	return func(ctx sdk.Context, tx sdk.Tx) (fee sdk.Coins, isEvm bool, from string, to *ethcmn.Address) {
		if evmTx, ok := tx.(*evmtypes.MsgEthereumTx); ok {
			isEvm = true
			_ = evmTx.VerifySig(evmTx.ChainID(), ctx.BlockHeight())
			from = evmTx.BaseTx.From
			to = evmTx.To()
			go func() {
				ak.GetAccount(ctx, evmTx.AccountAddress())
				if to != nil {
					ak.GetAccount(ctx, to.Bytes())
					ek.GetCode(ctx, *to)
				}
			}()

		}
		if feeTx, ok := tx.(authante.FeeTx); ok {
			fee = feeTx.GetFee()
		}

		return
	}
}

// fixLogForParallelTxHandler fix log for parallel tx
func fixLogForParallelTxHandler(ek *evm.Keeper) sdk.LogFix {
	return func(logIndex []int, anteErrs []error) (logs [][]byte) {
		return ek.FixLog(logIndex, anteErrs)
	}
}
