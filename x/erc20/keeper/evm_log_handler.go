package keeper

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
	"github.com/okex/exchain/x/erc20/types"
	evmtypes "github.com/okex/exchain/x/evm/types"
)

var (
	_ evmtypes.EvmLogHandler = SendToIbcEventHandler{}
)

const (
	SendToIbcEventName = "__OkcSendToIbc"
)

// SendToIbcEvent represent the signature of
// `event __OkcSendToIbc(string recipient, uint256 amount)`
var SendToIbcEvent abi.Event

func init() {
	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	stringType, _ := abi.NewType("string", "", nil)

	SendToIbcEvent = abi.NewEvent(
		SendToIbcEventName,
		SendToIbcEventName,
		false,
		abi.Arguments{abi.Argument{
			Name:    "sender",
			Type:    addressType,
			Indexed: false,
		}, abi.Argument{
			Name:    "recipient",
			Type:    stringType,
			Indexed: false,
		}, abi.Argument{
			Name:    "amount",
			Type:    uint256Type,
			Indexed: false,
		}},
	)
}

type SendToIbcEventHandler struct {
	Keeper
}

func NewSendToIbcEventHandler(k Keeper) *SendToIbcEventHandler {
	return &SendToIbcEventHandler{k}
}

// EventID Return the id of the log signature it handles
func (h SendToIbcEventHandler) EventID() common.Hash {
	return SendToIbcEvent.ID
}

// Handle Process the log
func (h SendToIbcEventHandler) Handle(ctx sdk.Context, contract common.Address, data []byte) error {
	h.Logger(ctx).Info("trigger evm event", "event", SendToIbcEvent.Name, "contract", contract)
	// first confirm that the contract address and denom are registered,
	// to avoid unpacking any contract '__OkcSendToIbc' event, which consumes performance
	denom, found := h.Keeper.GetDenomByContract(ctx, contract)
	if !found {
		return fmt.Errorf("contract %s is not connected to native token", contract)
	}
	if !types.IsValidIBCDenom(denom) {
		return fmt.Errorf("the native token associated with the contract %s is not an ibc voucher", contract)
	}

	unpacked, err := SendToIbcEvent.Inputs.Unpack(data)
	if err != nil {
		// log and ignore
		h.Keeper.Logger(ctx).Info("log signature matches but failed to decode")
		return nil
	}

	contractAddr := sdk.AccAddress(contract.Bytes())
	sender := sdk.AccAddress(unpacked[0].(common.Address).Bytes())
	recipient := unpacked[1].(string)
	amount := sdk.NewIntFromBigInt(unpacked[2].(*big.Int))
	amountDec := sdk.NewDecFromIntWithPrec(amount, sdk.Precision)
	vouchers := sdk.NewCoins(sdk.NewCoin(denom, amountDec))

	// 1. transfer IBC coin to user so that he will be the refunded address if transfer fails
	if err = h.bankKeeper.SendCoins(ctx, contractAddr, sender, vouchers); err != nil {
		return err
	}
	// 2. Initiate IBC transfer from sender account
	if err = h.Keeper.IbcTransferVouchers(ctx, sender.String(), recipient, vouchers); err != nil {
		return err
	}
	return nil
}
