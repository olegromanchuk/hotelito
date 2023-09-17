package boltstore

import (
	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
	"os"
	"testing"
)

var (
	testDbName       = "test_secrets.db"
	testDbBucketName = "test_bucket"
)

func setup() *BoltDBStore {
	store, err := Initialize(testDbName, testDbBucketName)
	if err != nil {
		panic(err)
	}
	return store
}

func teardown(store *BoltDBStore) {
	store.Close()
	os.Remove(testDbName)
}

func TestStoreAccessToken(t *testing.T) {

	tests := []struct {
		name       string
		token      string
		bucketName string
		expectErr  bool
	}{
		{"Valid Token", "test_access_token", "test_bucket", false},
		{"Empty bucket name", "test_access_token", "", true},
		{"Empty token", "", "test_bucket", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()
			defer teardown(store)
			store.BucketName = tt.bucketName

			err := store.StoreAccessToken(tt.token)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}

}

func TestRetrieveAccessToken(t *testing.T) {

	tests := []struct {
		name        string
		token       string
		bucketName  string
		expectValue string
		expectErr   bool
	}{
		{"Valid Token", "test_access_token", "test_bucket", "test_access_token", false},
		{"Empty bucket name", "test_access_token", "", "", false},
		{"Empty token", "", "test_bucket", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()
			store.BucketName = tt.bucketName

			_ = store.StoreAccessToken(tt.token)
			token, err := store.RetrieveAccessToken()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValue, token)
			teardown(store)
		})
	}
}

func TestStoreRefreshToken(t *testing.T) {

	tests := []struct {
		name       string
		token      string
		bucketName string
		expectErr  bool
	}{
		{"Valid Token", "test_access_token", "test_bucket", false},
		{"Empty bucket name", "test_access_token", "", true},
		{"Empty token", "", "test_bucket", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()
			defer teardown(store)
			store.BucketName = tt.bucketName

			err := store.StoreRefreshToken(tt.token)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestRetrieveRefreshToken(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		bucketName  string
		expectValue string
		expectErr   bool
	}{
		{"Valid Token", "test_access_token", "test_bucket", "test_access_token", false},
		{"Empty bucket name", "test_access_token", "", "", false},
		{"Empty token", "", "test_bucket", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()
			store.BucketName = tt.bucketName

			_ = store.StoreRefreshToken(tt.token)
			token, err := store.RetrieveRefreshToken()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValue, token)
			teardown(store)
		})
	}
}

func TestStoreOauthState(t *testing.T) {

	tests := []struct {
		name        string
		token       string
		expectValue string
		expectErr   bool
	}{
		{"Valid Token", "test_access_token", "test_access_token", false},
		{"Empty token and bucket", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()

			err := store.StoreOauthState(tt.token)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			teardown(store)
		})
	}
}

func TestRetrieveOauthState(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		expectValue string
		expectErr   bool
	}{
		{"Valid Token", "test_access_token", "test_access_token", false},
		{"Empty token and bucket", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()

			_ = store.StoreOauthState(tt.token)
			token, err := store.RetrieveOauthState(tt.token)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValue, token)
			teardown(store)
		})
	}
}

func TestRetrieveVar(t *testing.T) {

	tests := []struct {
		name        string
		VarName     string
		VarValue    string
		bucketName  string
		expectValue string
		expectErr   bool
	}{
		{"Valid value", "var_name_A", "var_value_A", "test_bucket", "var_value_A", false},
		{"Empty  bucket", "var_name_A", "var_value_A", "", "", false},
		{"Empty  name", "", "", "test_bucket", "", false},
		{"Empty  value", "var_name_A", "", "test_bucket", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setup()
			store.BucketName = tt.bucketName

			err := store.Db.Update(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucketIfNotExists([]byte("test_bucket"))
				if err != nil {
					return err
				}
				return bucket.Put([]byte("var_name_A"), []byte(tt.VarValue))
			})
			assert.Nil(t, err)

			val, err := store.RetrieveVar(tt.VarName)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectValue, val)

			teardown(store)
		})
	}

}

func TestInitializeAndClose(t *testing.T) {
	store, err := Initialize(testDbName, testDbBucketName)
	assert.Nil(t, err)

	err = store.Close()
	assert.Nil(t, err)
	os.Remove(testDbName)
}
