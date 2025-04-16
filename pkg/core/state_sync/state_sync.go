package statesync

import (
	"github.com/AudiusProject/audiusd/pkg/core/common"
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/core/db"
)

type StateSync struct {
	config *config.Config
	logger *common.Logger
	db     *db.Queries
}

func NewStateSync(config *config.Config, logger *common.Logger, db *db.Queries) *StateSync {
	return &StateSync{db: db, config: config, logger: logger}
}
