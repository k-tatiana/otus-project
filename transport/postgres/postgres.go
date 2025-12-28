package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DB holds master and slave database connection pools.
type DB struct {
	Master *pgxpool.Pool
	Slave  *pgxpool.Pool
}

// New initializes and returns database connections.
func New(ctx context.Context, master *string, slave *string, enableSlave bool, logger *zap.Logger) (*DB, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	var err error
	db := &DB{}

	// Initialize master database connection
	db.Master, err = pgxpool.New(ctx, *master)
	if err != nil {
		return nil, err
	}

	// Test master connection
	if err = db.Master.Ping(ctx); err != nil {
		db.Master.Close()
		return nil, err
	}
	logger.Info("Master database connection established")

	// Initialize slave database connection if configured
	if enableSlave && slave != nil {
		db.Slave, err = pgxpool.New(ctx, *slave)
		if err != nil {
			db.Master.Close()
			return nil, err
		}

		// Test slave connection
		if err = db.Slave.Ping(ctx); err != nil {
			db.Master.Close()
			db.Slave.Close()
			return nil, err
		}
		logger.Info("Slave database connection established")
	} else {
		// Use master as slave if no slave is configured
		db.Slave = db.Master
		logger.Warn("Using master database for read operations (no slave configured)")
	}

	return db, nil
}

// Close closes all database connection pools.
func (db *DB) Close(logger *zap.Logger) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if db.Master != nil {
		db.Master.Close()
		logger.Info("Master database connection closed")
	}
	if db.Slave != nil && db.Slave != db.Master {
		db.Slave.Close()
		logger.Info("Slave database connection closed")
	}
}
