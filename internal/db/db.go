package db

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/yourorg/zone-names/internal/models"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

type Config struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    SSLMode  string
}

type Database struct {
    DB *gorm.DB
}

func NewDatabase(cfg Config) (*Database, error) {
    dsn := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
    )

    newLogger := logger.New(
        log.New(os.Stdout, "\r\n", log.LstdFlags),
        logger.Config{
            SlowThreshold:             time.Second,
            LogLevel:                  logger.Info,
            IgnoreRecordNotFoundError: true,
            Colorful:                  true,
        },
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: newLogger,
    })

    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    err = db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error
    if err != nil {
        return nil, fmt.Errorf("failed to create uuid extension: %w", err)
    }

    err = db.AutoMigrate(
        &models.DomainLabel{},
        &models.Tag{},
        &models.DomainLabelTag{},
    )

    if err != nil {
        return nil, fmt.Errorf("failed to auto-migrate models: %w", err)
    }

    return &Database{DB: db}, nil
}

func (d *Database) Close() error {
    sqlDB, err := d.DB.DB()
    if err != nil {
        return fmt.Errorf("error getting sql.DB: %w", err)
    }
    return sqlDB.Close()
}
