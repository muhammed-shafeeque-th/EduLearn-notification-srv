package database

import (
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/database/gorm/models"
	log "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	// "moul.io/zapgorm2"
)

type DB struct {
	db     *gorm.DB
	logger ports.LoggerService
}

func NewDB(dsn string, logger ports.LoggerService) (*DB, error) {
	// zapLogger := zapgorm2.New(logger)
	// zapLogger.SetAsDefault()
	// zapLogger.LogLevel = 2 // only log errors

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Logger:                 zapLogger,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		logger.Error("failed to connect database", log.Error(err))
		return nil, err
	}
	logger.Info("database connected")
	return &DB{db: db, logger: logger}, nil
}

func (d *DB) Gorm() *gorm.DB {
	return d.db
}

func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		d.logger.Error("failed to get sql DB", log.Error(err))
		return err
	}
	if err := sqlDB.Close(); err != nil {
		d.logger.Error("failed to close DB", log.Error(err))
		return err
	}
	d.logger.Info("database closed")
	return nil
}

func (r *DB) AutoMigrate() error {
	if err := r.db.AutoMigrate(&models.NotificationModel{}, &models.ProcessedNotificationModel{}); err != nil {
		r.logger.Error("failed to auto-migrate", log.Error(err))
		return err
	}
	return nil
}
