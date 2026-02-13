package app

import (
	"errors"
	"time"

	errutils "auction-platform/pkg/errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	log "github.com/sirupsen/logrus"
)

const (
	defaultAttempts = 10
	defaultTimeout  = time.Second
)

func Migrate(pgUrl string) {
	pgUrl += "?sslmode=disable"
	log.Infof("Migrate %s", pgUrl)

	var (
		connAttempts = defaultAttempts
		err          error
		mgrt         *migrate.Migrate
	)

	for connAttempts > 0 {
		mgrt, err = migrate.New("file://migrations", pgUrl)
		if err == nil {
			break
		}
		time.Sleep(defaultTimeout)
		log.Infof("Postgres trying to connect, attempts left: %d", connAttempts)
		connAttempts--
	}

	if err != nil {
		panic(errutils.WrapPathErr(err))
	}
	defer mgrt.Close()

	if err = mgrt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		panic(errutils.WrapPathErr(err))
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Info("Migration no change")
		return
	}

	log.Info("Migration successful up")
}
