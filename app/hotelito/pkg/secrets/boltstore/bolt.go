package boltstore

import bolt "go.etcd.io/bbolt"

const (
	dbFile     = "secrets.db"
	bucketName = "cloudbeds_creds"
)

type BoltDBStore struct {
	Db *bolt.DB
}

func (s *BoltDBStore) StoreAccessToken(token string) error {
	return s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte("access_token"), []byte(token))
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *BoltDBStore) StoreRefreshToken(token string) error {
	return s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte("refresh_token"), []byte(token))
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *BoltDBStore) RetrieveAccessToken() (string, error) {
	var token string
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}

		tokenBytes := bucket.Get([]byte("access_token"))
		if tokenBytes == nil {
			return nil
		}

		token = string(tokenBytes)
		return nil
	})

	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *BoltDBStore) RetrieveRefreshToken() (string, error) {
	var token string
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}

		tokenBytes := bucket.Get([]byte("refresh_token"))
		if tokenBytes == nil {
			return nil
		}

		token = string(tokenBytes)
		return nil
	})

	if err != nil {
		return "", err
	}

	return token, nil
}

func Initialize() (*BoltDBStore, error) {
	dbref, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltDBStore{Db: dbref}, nil
}

func (s *BoltDBStore) Close() error {
	return s.Db.Close()
}
