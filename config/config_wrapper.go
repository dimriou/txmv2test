package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tomldecode "github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-evm/pkg/assets"
	"github.com/smartcontractkit/chainlink-evm/pkg/config"
	"github.com/smartcontractkit/chainlink-evm/pkg/config/toml"
	"github.com/smartcontractkit/chainlink-evm/pkg/types"
)

// This wrapper is required because of the way Gas Estimator components expect configs.
// Instead of passing down a struct with values, we need to imlpement an interface with
// the required methods.
type GasEstimator struct {
	EIP1559DynamicFeesF bool
	BumpPercentF        uint16
	BumpThresholdF      uint64
	BumpTxDepthF        uint32
	BumpMinF            *assets.Wei
	FeeCapDefaultF      *assets.Wei
	LimitDefaultF       uint64
	LimitMaxF           uint64
	LimitMultiplierF    float32
	LimitTransferF      uint64
	PriceDefaultF       *assets.Wei
	TipCapDefaultF      *assets.Wei
	TipCapMinF          *assets.Wei
	PriceMaxF           *assets.Wei
	PriceMinF           *assets.Wei
	ModeF               string
	EstimateLimitF      bool
	SenderAddressF      *types.EIP55Address
	FeeHistoryF         *FeeHistory
	BlockHistoryF       *BlockHistory
}

func (g GasEstimator) PriceMaxKey(common.Address) *assets.Wei {
	return g.PriceMaxF
}

func (g GasEstimator) EIP1559DynamicFees() bool {
	return g.EIP1559DynamicFeesF
}

func (b GasEstimator) BumpPercent() uint16 {
	return b.BumpPercentF
}

func (g GasEstimator) BumpThreshold() uint64 {
	return g.BumpThresholdF
}

func (g GasEstimator) BumpTxDepth() uint32 {
	return g.BumpTxDepthF
}
func (g GasEstimator) BumpMin() *assets.Wei {
	return g.BumpMinF
}

func (g GasEstimator) TipCapMin() *assets.Wei {
	return g.TipCapMinF
}

func (g GasEstimator) PriceMax() *assets.Wei {
	return g.PriceMaxF
}

func (g GasEstimator) PriceMin() *assets.Wei {
	return g.PriceMinF
}

func (g GasEstimator) Mode() string {
	return g.ModeF
}

func (g GasEstimator) PriceDefault() *assets.Wei {
	return g.PriceDefaultF
}

func (g GasEstimator) TipCapDefault() *assets.Wei {
	return g.TipCapDefaultF
}

func (g GasEstimator) FeeCapDefault() *assets.Wei {
	return g.FeeCapDefaultF
}

func (g GasEstimator) LimitDefault() uint64 {
	return g.LimitDefaultF
}

func (g GasEstimator) LimitMax() uint64 {
	return g.LimitMaxF
}

func (g GasEstimator) LimitMultiplier() float32 {
	return g.LimitMultiplierF
}

func (g GasEstimator) LimitTransfer() uint64 {
	return g.LimitTransferF
}

func (g GasEstimator) EstimateLimit() bool {
	return g.EstimateLimitF
}

func (g GasEstimator) SenderAddress() *types.EIP55Address {
	return g.SenderAddressF
}

// -------------------------------
func (g GasEstimator) DAOracle() config.DAOracle {
	return &DAOracle{}
}

type DAOracle struct {
	OracleTypeF             *toml.DAOracleType
	OracleAddressF          *types.EIP55Address
	CustomGasPriceCalldataF *string
}

func (o DAOracle) OracleType() *toml.DAOracleType {
	return o.OracleTypeF
}

func (o DAOracle) OracleAddress() *types.EIP55Address {
	return o.OracleAddressF
}

func (o DAOracle) CustomGasPriceCalldata() *string {
	return o.CustomGasPriceCalldataF
}

// -------------------------------
func (g GasEstimator) LimitJobType() config.LimitJobType {
	return nil
}

// -------------------------------
func (g GasEstimator) FeeHistory() config.FeeHistory {
	return g.FeeHistoryF
}

type FeeHistory struct {
	CacheTimeoutF time.Duration
}

func (b FeeHistory) CacheTimeout() time.Duration {
	return b.CacheTimeoutF
}

// -------------------------------
func (g GasEstimator) BlockHistory() config.BlockHistory {
	return g.BlockHistoryF
}

type BlockHistory struct {
	BatchSizeF                 uint32
	BlockHistorySizeF          uint16
	BlockDelayF                uint16
	CheckInclusionBlocksF      uint16
	CheckInclusionPercentileF  uint16
	EIP1559FeeCapBufferBlocksF uint16
	TransactionPercentileF     uint16
}

func (b BlockHistory) BatchSize() uint32 {
	return b.BatchSizeF
}

func (b BlockHistory) BlockHistorySize() uint16 {
	return b.BlockHistorySizeF
}

func (b BlockHistory) BlockDelay() uint16 {
	return b.BlockDelayF
}

func (b BlockHistory) CheckInclusionBlocks() uint16 {
	return b.CheckInclusionBlocksF
}

func (b BlockHistory) CheckInclusionPercentile() uint16 {
	return b.CheckInclusionPercentileF
}

func (b BlockHistory) EIP1559FeeCapBufferBlocks() uint16 {
	return b.EIP1559FeeCapBufferBlocksF
}

func (b BlockHistory) TransactionPercentile() uint16 {
	return b.TransactionPercentileF
}

// AppConfig holds the application configuration loaded from TOML and environment variables
type AppConfig struct {
	RPC         string `toml:"rpc"`
	PrivateKey  string `toml:"private_key"`
	FromAddress string `toml:"from_address"`
}

// LoadAppConfig loads the application configuration from a TOML file and environment variables.
// Environment variables take precedence over TOML values.
// If configPath is empty, it defaults to "config.toml"
func LoadAppConfig(configPath string) (*AppConfig, error) {
	if configPath == "" {
		configPath = "env.toml"
	}

	cfg := &AppConfig{}

	// Read TOML file if it exists
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err == nil {
		if err := tomldecode.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse TOML config: %w", err)
		}
	}

	if envRPC := os.Getenv("RPC"); envRPC != "" {
		cfg.RPC = envRPC
	} else {
		return nil, fmt.Errorf("RPC is required (set in env.toml)")
	}
	if envPrivateKey := os.Getenv("PRIVATE_KEY"); envPrivateKey != "" {
		cfg.PrivateKey = envPrivateKey
	} else {
		return nil, fmt.Errorf("PRIVATE_KEY is required (set in env.toml)")
	}
	if envFromAddress := os.Getenv("FROM_ADDRESS"); envFromAddress != "" {
		cfg.FromAddress = envFromAddress
	} else {
		return nil, fmt.Errorf("FROM_ADDRESS is required (set in env.toml)")
	}

	return cfg, nil
}
