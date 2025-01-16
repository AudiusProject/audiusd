package pages

import (
	"github.com/AudiusProject/audiusd/pkg/core/config"
	"github.com/AudiusProject/audiusd/pkg/core/console/views/components"
	"github.com/AudiusProject/audiusd/pkg/core/console/views/layout"
)

type Pages struct {
	config     *config.Config
	components *components.Components
	layout     *layout.Layout
}

func NewPages(config *config.Config, baseUrl string) *Pages {
	return &Pages{
		config:     config,
		components: components.NewComponents(config, baseUrl),
		layout:     layout.NewLayout(config, baseUrl),
	}
}
