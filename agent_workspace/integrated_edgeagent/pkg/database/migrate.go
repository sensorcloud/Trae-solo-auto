package database

import (
	"github.com/ywtech/edgeagent-hub/internal/agent"
	"github.com/ywtech/edgeagent-hub/internal/billing"
	"github.com/ywtech/edgeagent-hub/internal/coordination"
	"github.com/ywtech/edgeagent-hub/internal/energy"
	"github.com/ywtech/edgeagent-hub/internal/iot"
	"github.com/ywtech/edgeagent-hub/internal/market"
	"github.com/ywtech/edgeagent-hub/internal/monitor"
	"github.com/ywtech/edgeagent-hub/internal/user"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&user.User{},
		&agent.Agent{},
		&market.Asset{},
		&market.Order{},
		&energy.PowerSource{},
		&energy.Storage{},
		&coordination.ScheduleHistory{},
		&iot.Device{},
		&iot.Telemetry{},
		&billing.Bill{},
		&billing.UsageRecord{},
		&monitor.Alert{},
	)
}

func InitModules(db *gorm.DB) {
	user.InitDB(db)
	agent.InitDB(db)
	market.InitDB(db)
	energy.InitDB(db)
	coordination.InitDB(db)
	iot.InitDB(db)
	billing.InitDB(db)
	monitor.InitDB(db)
}