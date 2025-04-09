package logger

import (
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
		AxiomDataset: os.Getenv("AUDIUSD_AXIOM_DATASET"),
		AxiomToken:   os.Getenv("AUDIUSD_AXIOM_TOKEN"),
		ZapLevel:     os.Getenv("AUDIUSD_LOG_LEVEL"),
	}
}

func (c *LoggerConfig) SetAxiomDataset(dataset string) {
	c.AxiomDataset = dataset
}

func (c *LoggerConfig) AxiomEnabled() bool {
	return c.AxiomToken != ""
}

func (c *LoggerConfig) CreateLogger() (*zap.Logger, error) {
	cores := []zapcore.Core{}

	if c.AxiomEnabled() {
		axiomCore, err := adapter.New(
			adapter.SetDataset(c.AxiomDataset),
			adapter.SetClientOptions(axiom.SetAPITokenConfig(c.AxiomToken)),
		)
		if err != nil {
			return nil, err
		}
		cores = append(cores, axiomCore)
	}

	// Stdout core (human-readable or JSON)
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	stdoutCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	cores = append(cores, stdoutCore)

	combinedCore := zapcore.NewTee(cores...)
	logger := zap.New(combinedCore)

	return logger, nil
}
