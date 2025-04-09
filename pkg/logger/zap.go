package logger

import (
	"fmt"
	"os"

	"github.com/axiomhq/axiom-go/axiom"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	adapter "github.com/axiomhq/axiom-go/adapters/zap"
)

type LoggerConfig struct {
	AxiomDataset string
	AxiomToken   string
	ZapLevel     string
}

func ReadLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		AxiomDataset: "core-dev",
		AxiomToken:   "xaat-349f57c1-82e2-4f3e-9e0d-35a24a726a3b",
		ZapLevel:     "debug",
	}
}

func (c *LoggerConfig) SetAxiomDataset(dataset string) {
	c.AxiomDataset = dataset
}

func (c *LoggerConfig) SetLogLevel(level string) {
	c.ZapLevel = level
}

func (c *LoggerConfig) AxiomEnabled() bool {
	return c.AxiomToken != ""
}

func (c *LoggerConfig) CreateLogger() (*zap.Logger, error) {
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	zapLevel, err := zapcore.ParseLevel(c.ZapLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse zap level: %v", err)
	}
	stdoutCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapLevel)

	axiomCore, err := adapter.New(
		adapter.SetDataset(c.AxiomDataset),
		adapter.SetClientOptions(axiom.SetAPITokenConfig(c.AxiomToken)),
	)
	if err != nil {
		return nil, err
	}
	combinedCore := zapcore.NewTee(axiomCore, stdoutCore)
	return zap.New(combinedCore), nil
}
