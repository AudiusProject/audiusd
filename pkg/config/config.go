package config

import (
	"time"

	cconfig "github.com/cometbft/cometbft/config"
)

// Config is the top level config the audiusd and cometbft services
type Config struct {
	cconfig.Config `mapstructure:",squash"`
	Audiusd        *AudiusdConfig `mapstructure:"audiusd"`
}

func DefaultConfig() *Config {
	return &Config{
		Config:  *DefaultCometBFTConfig(),
		Audiusd: DefaultAudiusdConfig(),
	}
}

func DefaultCometBFTConfig() *cconfig.Config {
	defaultConfig := cconfig.DefaultConfig()

	// indexer
	defaultConfig.TxIndex.Indexer = "null"

	// mempool
	defaultConfig.Mempool.MaxTxsBytes = 10485760
	defaultConfig.Mempool.MaxTxBytes = 307200
	defaultConfig.Mempool.Size = 30000

	// consensus
	defaultConfig.Mempool.Recheck = false
	defaultConfig.Mempool.Broadcast = false
	defaultConfig.Consensus.TimeoutCommit = 400 * time.Millisecond
	defaultConfig.Consensus.TimeoutPropose = 400 * time.Millisecond
	defaultConfig.Consensus.TimeoutProposeDelta = 75 * time.Millisecond
	defaultConfig.Consensus.TimeoutPrevote = 300 * time.Millisecond
	defaultConfig.Consensus.TimeoutPrevoteDelta = 75 * time.Millisecond
	defaultConfig.Consensus.TimeoutPrecommit = 300 * time.Millisecond
	defaultConfig.Consensus.TimeoutPrecommitDelta = 75 * time.Millisecond
	defaultConfig.Consensus.CreateEmptyBlocks = true
	defaultConfig.Consensus.CreateEmptyBlocksInterval = 1 * time.Second

	// p2p
	defaultConfig.P2P.PexReactor = true
	defaultConfig.P2P.AddrBookStrict = true
	defaultConfig.P2P.MaxNumOutboundPeers = 50
	defaultConfig.P2P.MaxNumInboundPeers = 200
	defaultConfig.P2P.AllowDuplicateIP = true
	defaultConfig.P2P.FlushThrottleTimeout = 50 * time.Millisecond
	defaultConfig.P2P.SendRate = 5120000
	defaultConfig.P2P.RecvRate = 5120000
	defaultConfig.P2P.HandshakeTimeout = 3 * time.Second
	defaultConfig.P2P.DialTimeout = 5 * time.Second
	defaultConfig.P2P.PersistentPeersMaxDialPeriod = 15 * time.Second

	// pruning
	defaultConfig.Storage.Compact = true
	defaultConfig.Storage.DiscardABCIResponses = true
	defaultConfig.GRPC.Privileged = &cconfig.GRPCPrivilegedConfig{
		ListenAddress: "unix:///tmp/cometbft.privileged.sock",
		PruningService: &cconfig.GRPCPruningServiceConfig{
			Enabled: true,
		},
	}
	defaultConfig.Storage.Pruning.DataCompanion = &cconfig.DataCompanionPruningConfig{
		Enabled: true,
	}

	return defaultConfig
}

// AudiusdConfig is the top level config for the audiusd service
type AudiusdConfig struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`

	// Options for services
	RPC      *RPCConfig      `mapstructure:"rpc"`
	Core     *CoreConfig     `mapstructure:"core"`
	Storage  *StorageConfig  `mapstructure:"storage"`
	Registry *RegistryConfig `mapstructure:"registry"`
	Etl      *EtlConfig      `mapstructure:"etl"`
	Console  *ConsoleConfig  `mapstructure:"console"`
	System   *SystemConfig   `mapstructure:"system"`
}

func DefaultAudiusdConfig() *AudiusdConfig {
	return &AudiusdConfig{
		BaseConfig: *DefaultBaseConfig(),
		RPC:        DefaultRPCConfig(),
		Core:       DefaultCoreConfig(),
		Storage:    DefaultStorageConfig(),
		Registry:   DefaultRegistryConfig(),
		Etl:        DefaultEtlConfig(),
		Console:    DefaultConsoleConfig(),
		System:     DefaultSystemConfig(),
	}
}

type BaseConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	PrivKey  string `mapstructure:"priv_key"`
}

func DefaultBaseConfig() *BaseConfig {
	return &BaseConfig{
		Endpoint: "http://0.0.0.0:8080",
		PrivKey:  "",
	}
}

type RPCConfig struct {
	HttpPort      string `mapstructure:"http_port"`
	HttpsPort     string `mapstructure:"https_port"`
	TlsEnabled    bool   `mapstructure:"tls_enabled"`
	TlsSelfSigned bool   `mapstructure:"tls_self_signed"`
	GrpcEnabled   bool   `mapstructure:"grpc_enabled"`
	GrpcPort      string `mapstructure:"grpc_port"`
}

func DefaultRPCConfig() *RPCConfig {
	return &RPCConfig{
		HttpPort:      "80",
		HttpsPort:     "443",
		TlsEnabled:    false,
		TlsSelfSigned: false,
		GrpcEnabled:   false,
		GrpcPort:      "50051",
	}
}

type CoreConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

func DefaultCoreConfig() *CoreConfig {
	return &CoreConfig{
		LogLevel: "info",
	}
}

type StorageConfig struct {
	LogLevel string `mapstructure:"log_level"`
	Enabled  bool   `mapstructure:"enabled"`
}

func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Enabled:  false,
		LogLevel: "info",
	}
}

type RegistryConfig struct {
	LogLevel    string             `mapstructure:"log_level"`
	EthRegistry *EthRegistryConfig `mapstructure:"eth"`
}

func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		LogLevel: "info",
	}
}

type EthRegistryConfig struct {
	RPCUrl            string `mapstructure:"rpc_url"`
	RPCEnabled        bool   `mapstructure:"rpc_enabled"`
	ETHRPCEnabled     bool   `mapstructure:"eth_rpc_enabled"`
	PollingIntervalMS int    `mapstructure:"polling_interval_ms"`
	RegistryAddress   string `mapstructure:"registry_address"`
}

func DefaultEthRegistryConfig() *EthRegistryConfig {

	return &EthRegistryConfig{
		RPCEnabled:        false,
		PollingIntervalMS: 1000,
		RegistryAddress:   "",
	}
}

type EtlConfig struct {
	LogLevel   string `mapstructure:"log_level"`
	RPCEnabled bool   `mapstructure:"rpc_enabled"`
	PgConn     string `mapstructure:"pg_conn"`
}

func DefaultEtlConfig() *EtlConfig {
	return &EtlConfig{
		RPCEnabled: true,
		LogLevel:   "info",
	}
}

type ConsoleConfig struct {
	LogLevel string `mapstructure:"log_level"`
	Enabled  bool   `mapstructure:"enabled"`
}

func DefaultConsoleConfig() *ConsoleConfig {
	return &ConsoleConfig{
		Enabled:  true,
		LogLevel: "info",
	}
}

type SystemConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

func DefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		LogLevel: "info",
	}
}
