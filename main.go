package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-evm/pkg/assets"
	"github.com/smartcontractkit/chainlink-evm/pkg/gas"

	"github.com/smartcontractkit/chainlink-evm/pkg/txm"
	"github.com/smartcontractkit/chainlink-evm/pkg/txm/clientwrappers"
	"github.com/smartcontractkit/chainlink-evm/pkg/txm/storage"
	"github.com/smartcontractkit/chainlink-evm/pkg/txm/types"

	"github.com/dimriou/txmv2test/config"
)

func main() {
	envVars, err := config.LoadAppConfig("env.toml")
	if err != nil || envVars == nil {
		log.Fatal(err)
	}

	// Configs for Ethereum Sepolia
	appConfigs := config.GasEstimator{
		EIP1559DynamicFeesF: true,
		BumpPercentF:        20,
		BumpThresholdF:      3,
		LimitDefaultF:       30000,
		LimitMultiplierF:    1,
		PriceMaxF:           assets.GWei(700),
		ModeF:               "FeeHistory",
		FeeHistoryF: &config.FeeHistory{
			CacheTimeoutF: 11 * time.Second,
		},
		BlockHistoryF: &config.BlockHistory{
			BlockHistorySizeF:      4,
			TransactionPercentileF: 55,
		},
	}

	// Init Logger.
	lggr, err := logger.NewWith(func(cfg *zap.Config) {
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000000000")
		cfg.Level.SetLevel(zap.DebugLevel)
		cfg.Development = true
		cfg.OutputPaths = []string{"stdout"}
	})

	if err != nil {
		log.Fatal(err)
	}

	// Dial Ethereum Client.
	c, err := ethclient.Dial(envVars.RPC)
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

	// Init estimator (only FeeHistory estimator is supported outside the Core node).
	estimator, err := initGasEstimator(lggr, client, chainID, appConfigs)
	if err != nil {
		lggr.Fatal("Error during estimator init", err)
		return
	}

	// Init Dummy Keystore
	keystore := txm.NewKeystore(chainID)
	if err := keystore.Add(envVars.PrivateKey); err != nil {
		lggr.Fatal("Error adding key to keystore", err)
		return
	}

	// AttemptBuilder creates new attempts using Gas Estimator's estimates and signs them using the Keystore
	ab := txm.NewAttemptBuilder(appConfigs.PriceMaxKey, estimator, keystore, appConfigs.LimitDefaultF)

	// Init InMemory storage instead of a Database.
	store := storage.NewInMemoryStoreManager(lggr, chainID)
	fromAddress := common.HexToAddress(envVars.FromAddress)
	if err := store.Add(fromAddress); err != nil {
		lggr.Fatal("Error adding address to InMemory store", err)
		return
	}

	// Init TXMv2
	txmConfig := txm.Config{
		EIP1559:             appConfigs.EIP1559DynamicFees(),
		BlockTime:           appConfigs.FeeHistory().CacheTimeout(),
		RetryBlockThreshold: uint16(appConfigs.BumpThreshold()),
		EmptyTxLimitDefault: appConfigs.LimitDefaultF,
	}
	stuckConfig := txm.StuckTxDetectorConfig{
		BlockTime:             appConfigs.FeeHistory().CacheTimeout(),
		StuckTxBlockThreshold: 3,
	}
	st := txm.NewStuckTxDetector(lggr, "", stuckConfig)
	//eh := dualbroadcast.NewErrorHandler()
	txm := txm.NewTxm(lggr, chainID, client, ab, store, st, txmConfig, keystore, nil)
	err = txm.Start(context.Background())
	if err != nil {
		lggr.Fatal("Failed to start txm", err)
		return
	}
	// =======================Add your logic here==============================
	for range 5 {
		//Create request
		txRequest := types.TxRequest{
			ChainID:           chainID,
			FromAddress:       fromAddress,
			ToAddress:         common.HexToAddress("0x45BB36B79E02e59d3C49b863B31F530C991dd554"),
			Value:             big.NewInt(50),
			Data:              []byte{128, 100, 11},
			SpecifiedGasLimit: 40000,
		}

		_, err = txm.CreateTransaction(context.TODO(), &txRequest) // Transaction is created
		if err != nil {
			lggr.Fatal("Failed to create transaction", err)
			return
		}

	}
	txm.Trigger(fromAddress) // Trigger instantly triggers the TXM instead of waiting for the next cycle.
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
		fmt.Println("Unstarted: ", unstartedCount, " - Unconfirmed: ", unconfirmedCount)
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
	time.Sleep(3 * time.Second)
	return estimator, nil
}
