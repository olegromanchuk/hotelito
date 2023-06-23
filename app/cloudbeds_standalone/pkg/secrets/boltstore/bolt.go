package boltstore

import bolt "go.etcd.io/bbolt"

const (
	dbFile       = "secrets.db"
	accessToken  = "access_token"
	defaultToken = "" // Default token value when key is not found in the DB
)

var (
	db *bolt.DB
)

type BoltDBStore struct {
	Db *bolt.DB
}

func (s *BoltDBStore) StoreAccessToken(token string) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(accessToken))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(accessToken), []byte(token))
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *BoltDBStore) RetrieveAccessToken() (string, error) {
	var token string
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(accessToken))
		if bucket == nil {
			return nil
		}

		tokenBytes := bucket.Get([]byte(accessToken))
		if tokenBytes == nil {
			return nil
		}

		token = string(tokenBytes)
		return nil
	})

	if err != nil {
		return "", err
	}

	if token == "" {
		return defaultToken, nil
	}

	return token, nil
}

func New() (*BoltDBStore, error) {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltDBStore{db: db}, nil
}
