package database

import (
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/agent"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/billing"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/coordination"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/energy"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/iot"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/market"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/monitor"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/user"

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