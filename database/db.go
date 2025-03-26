package database

import (
	"database/sql"
	"errors"
	"time"

	"koth.cyber.cs.unh.edu/lib"
	_ "modernc.org/sqlite"
)

const (
	maxRetries = 5
	baseDelay  = 100 * time.Millisecond
)

var ErrBadData = errors.New("bad data")

var db *sql.DB

func open() (*sql.DB, error) {
	var err error

	for i := range maxRetries {
		db, err = sql.Open("sqlite", lib.Config.Database.File)
		if err == nil {
			break
		}

		time.Sleep(baseDelay * time.Duration(i))
	}

	return db, err
}

func Connect() error {
	var err error

	db, err = open()

	if err != nil {
		return err
	}

	if _, err = db.Exec(TEAMS_STATEMENT); err != nil {
		return err
	}

	if _, err = db.Exec(BLOBS_STATEMENT); err != nil {
		return err
	}

	return nil
}
