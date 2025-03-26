package database

import (
	"encoding/json"
	"errors"
)

var ErrBlobExists = errors.New("blob already exists")
var ErrBlobNotFound = errors.New("blob not found")

const BLOBS_STATEMENT = `CREATE TABLE IF NOT EXISTS blobs (
	name TEXT PRIMARY KEY NOT NULL,
	value TEXT NOT NULL
);`

const INSERT_BLOB_STATEMENT = `INSERT INTO blobs (name, value) VALUES (?, ?);`
const SELECT_BLOB_STATEMENT = `SELECT name, value FROM blobs WHERE name = ?;`
const DELETE_BLOB_STATEMENT = `DELETE FROM blobs WHERE name = ?;`
const SELECT_ALL_BLOBS_STATEMENT = `SELECT name, value FROM blobs;`
const UPDATE_BLOB_STATEMENT = `UPDATE blobs SET value = ? WHERE name = ?;`

type DBBlob struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (u *DBBlob) JSON() []byte {
	json, _ := json.Marshal(u)
	return json
}

func BlobExists(name string) bool {
	rows, err := QueuedQuery(SELECT_BLOB_STATEMENT, name)

	if err != nil {
		return false
	}

	defer rows.Close()
	return rows.Next()
}

func CreateBlob(name, value string) (*DBBlob, error) {
	if BlobExists(name) {
		return nil, ErrBlobExists
	}

	if err := QueuedExec(INSERT_BLOB_STATEMENT, name, value); err != nil {
		return nil, err
	}

	return GetBlob(name)
}

func GetBlob(name string) (*DBBlob, error) {
	rows, err := QueuedQuery(SELECT_BLOB_STATEMENT, name)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, ErrTeamNotFound
	}

	var team DBBlob
	if err := rows.Scan(&team.Name, &team.Value); err != nil {
		return nil, err
	}

	return &team, nil
}

func DeleteBlob(name string) error {
	return QueuedExec(DELETE_BLOB_STATEMENT, name)
}

func GetAllBlobs() ([]*DBBlob, error) {
	rows, err := QueuedQuery(SELECT_ALL_BLOBS_STATEMENT)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var blobs []*DBBlob
	for rows.Next() {
		var blob DBBlob
		if err := rows.Scan(&blob.Name, &blob.Value); err != nil {
			return nil, err
		}

		blobs = append(blobs, &blob)
	}

	return blobs, nil
}

func UpdateBlob(blob *DBBlob) error {
	return QueuedExec(UPDATE_BLOB_STATEMENT, blob.Value, blob.Name)
}
