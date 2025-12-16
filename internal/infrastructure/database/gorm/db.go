package database

import (
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/database/gorm/models"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

type DB struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDB(dsn string, logger *zap.Logger) (*DB, error) {
	zapLogger := zapgorm2.New(logger)
	zapLogger.SetAsDefault()
	zapLogger.LogLevel = 2 // only log errors

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 zapLogger,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		logger.Error("failed to connect database", zap.Error(err))
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
		d.logger.Error("failed to get sql DB", zap.Error(err))
		return err
	}
	if err := sqlDB.Close(); err != nil {
		d.logger.Error("failed to close DB", zap.Error(err))
		return err
	}
	d.logger.Info("database closed")
	return nil
}

func (r *DB) AutoMigrate() error {
	if err := r.db.AutoMigrate(&models.NotificationModel{}, &models.ProcessedNotificationModel{}); err != nil {
		r.logger.Error("failed to auto-migrate", zap.Error(err))
		return err
	}
	return nil
}