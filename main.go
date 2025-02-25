package main

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/assets"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/gas"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/txm"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/txm/clientwrappers"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/txm/storage"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/txm/types"
	"github.com/smartcontractkit/chainlink/v2/core/logger"

	"github.com/dimriou/txmv2test/config"
)

const (
	rpc               = "<REPLACE_WITH_RPC_URL>"     // RPC endpoint.
	privateKeyString  = "<REPLACE_WITH_PRIVATE_KEY>" // Private key without 0x prefix.
	fromAddressString = "<REPLACE_WITH_FROM_ADDRESS" // From address.
)

func main() {
	// Configs for Ethereum Sepolia
	gasConfig := config.GasEstimator{
		EIP1559DynamicFeesF: true,
		BumpPercentF:        20,
		LimitDefaultF:       30000,
		LimitMultiplierF:    1,
		PriceMaxF:           assets.GWei(700),
		ModeF:               "FeeHistory",
	}

	// Init Logger.
	lggrCfg := logger.Config{LogLevel: zapcore.DebugLevel}
	lggr, closeFn := lggrCfg.New()

	// Dial Ethereum Client.
	c, err := ethclient.Dial(rpc)
	if err != nil {
		lggr.Fatal("Couldn't connect to client!", err)
		return
	}

	// Create a client wrapper provided by the TXMv2 to comply with Core's MultiNode interface (MultiNode can still be used).
	client := clientwrappers.NewGethClient(c)
	chainID, err := client.ChainID(context.TODO())
	if err != nil {
		lggr.Fatal("Error fetching chainID", err)
		return
	}

	// Init estimator.
	estimator, err := initGasEstimator(lggr, client, chainID, gasConfig)
	if err != nil {
		lggr.Fatal("Error during estimator init", err)
		return
	}

	// Init Dummy Keystore
	keystore := txm.NewKeystore()
	if err := keystore.Add(privateKeyString); err != nil {
		lggr.Fatal("Error adding key to keystore", err)
		return
	}

	// AttemptBuilder creates new attempts using Gas Estimator's estimates and signs them using the Keystore
	ab := txm.NewAttemptBuilder(chainID, gasConfig.PriceMaxKey, estimator, keystore)

	// Init InMemory storage instead of a Database.
	store := storage.NewInMemoryStoreManager(lggr, chainID)
	fromAddress := common.HexToAddress(fromAddressString)
	if err := store.Add(fromAddress); err != nil {
		lggr.Fatal("Error adding address to InMemory store", err)
		return
	}

	// Init TXMv2
	txmConfig := txm.Config{
		EIP1559:             true,
		BlockTime:           12 * time.Second,
		RetryBlockThreshold: 3,
		EmptyTxLimitDefault: gasConfig.LimitDefaultF,
	}
	txm := txm.NewTxm(lggr, chainID, client, ab, store, nil, txmConfig, keystore)
	err = txm.Start(context.TODO())
	if err != nil {
		lggr.Fatal("Failed to start txm", err)
		return
	}

	// =======================Add your logic here==============================
	//Create request
	for i := 0; i <= 2; i++ {
		txRequest := types.TxRequest{
			ChainID:           chainID,
			FromAddress:       fromAddress,
			Value:             big.NewInt(50),
			Data:              []byte{128, 100, 11},
			SpecifiedGasLimit: gasConfig.LimitDefaultF,
		}

		_, err = txm.CreateTransaction(context.TODO(), &txRequest) // Transaction is created
		if err != nil {
			lggr.Fatal("Failed to create transaction", err)
			return
		}
		txm.Trigger(fromAddress)    // Trigger instantly triggers the TXM instead of waiting for the next cycle.
		time.Sleep(2 * time.Second) // Avoid spamming the RPC.
	}
	// ========================================================================

	// =========================Confirmation Loop==============================
	// This loop is not required but it's added in this example to confirm the broadcasted transactions
	// before we close TXMv2. TXMv2 is a service and will do that automatically as long as it's running.
	for {
		unstartedCount, err := store.CountUnstartedTransactions(fromAddress)
		if err != nil {
			lggr.Fatal("Failed to get unstarted transaction count", err)
			return
		}
		_, unconfirmedCount, _ := store.FetchUnconfirmedTransactionAtNonceWithCount(context.TODO(), 0, fromAddress)
		if unstartedCount == 0 && unconfirmedCount == 0 {
			break
		}
		time.Sleep(4 * time.Second)
	}
	if err := txm.Close(); err != nil {
		lggr.Fatal("Failed to close txm ", err)
	}
	if err := estimator.Close(); err != nil {
		lggr.Fatal("Failed to close estimator ", err)
	}
	if err := closeFn(); err != nil {
		lggr.Fatal("Failed to close logger ", err)
	}
	lggr.Debug("Thanks for using TXMv2!")
	// ========================================================================
}

func initGasEstimator(lggr logger.Logger, client *clientwrappers.GethClient, chainID *big.Int, gasConfig config.GasEstimator) (gas.EvmFeeEstimator, error) {
	estimator, err := gas.NewEstimator(lggr, client, "", chainID, gasConfig, nil)
	if err != nil {
		lggr.Fatal("Couldn't create estimator!", err)
		return nil, err
	}
	err = estimator.Start(context.TODO())
	if err != nil {
		lggr.Fatal("Couldn't start estimator!", err)
		return nil, err
	}
	// Committing a synchronization sin for the purposes of this test to make sure the estimator has up-to-date prices.
	time.Sleep(2 * time.Second)
	return estimator, nil
}
