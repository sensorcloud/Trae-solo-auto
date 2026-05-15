package database

import (
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dbType := viper.GetString("database.type")

	var db *gorm.DB
	var err error

	switch dbType {
	case "sqlite":
		dbPath := viper.GetString("database.path")
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	case "postgres":
		host := viper.GetString("database.host")
		port := viper.GetString("database.port")
		name := viper.GetString("database.name")
		user := viper.GetString("database.user")
		password := viper.GetString("database.password")

		dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + name + " sslmode=disable TimeZone=Asia/Shanghai"
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		db, err = gorm.Open(sqlite.Open("./data/edgeagent.db"), &gorm.Config{})
	}

	if err != nil {
		return nil, err
	}

	return db, nil
}