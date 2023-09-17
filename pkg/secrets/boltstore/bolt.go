package boltstore

import (
	bolt "go.etcd.io/bbolt"
)

type BoltDBStore struct {
	Db         *bolt.DB
	BucketName string
}

func (s *BoltDBStore) StoreAccessToken(token string) error {
	res := s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(s.BucketName))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte("access_token"), []byte(token))
		if err != nil {
			return err
		}

		return nil
	})
	return res
}

func (s *BoltDBStore) StoreRefreshToken(token string) error {
	return s.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(s.BucketName))
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
		bucket := tx.Bucket([]byte(s.BucketName))
		if bucket == nil {
			return nil //do not return error here!!! This is a valid case when refresh token is not set
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
		bucket := tx.Bucket([]byte(s.BucketName))
		if bucket == nil {
			return nil //do not return error here!!! This is a valid case when refresh token is not set
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

func (s *BoltDBStore) StoreOauthState(state string) error {
	return s.Db.Update(func(tx *bolt.Tx) error {
		// we need custom bucket "state" here to effectively delete if after the state is retrieved
		bucket, err := tx.CreateBucketIfNotExists([]byte(state))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte(state), []byte(state))
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *BoltDBStore) RetrieveOauthState(state string) (string, error) {
	var token string
	err := s.Db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(state))
		if bucket == nil {
			return nil //do not return error here!!! This is a valid case when refresh token is not set
		}

		tokenBytes := bucket.Get([]byte(state))
		if tokenBytes == nil {
			return nil
		}

		token = string(tokenBytes)
		return nil
	})

	if err != nil {
		return "", err
	}

	//clean up. Delete retrieved state
	err = s.Db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(state))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *BoltDBStore) RetrieveVar(varName string) (varValue string, err error) {
	err = s.Db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(s.BucketName))
		if bucket == nil {
			return nil //do not return error here!!q! This is a valid case when refresh token is not set
		}

		varValueBytes := bucket.Get([]byte(varName))
		if varValueBytes == nil {
			return nil
		}

		varValue = string(varValueBytes)
		return nil
	})

	if err != nil {
		return "", err
	}

	return varValue, nil
}

func Initialize(dbFileName, bucket_name string) (*BoltDBStore, error) {
	dbref, err := bolt.Open(dbFileName, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltDBStore{
		Db:         dbref,
		BucketName: bucket_name,
	}, nil
}

func (s *BoltDBStore) Close() error {
	return s.Db.Close()
}
