package secrets

type SecretsStore interface {
	StoreAccessToken(token string) error
	StoreRefreshToken(token string) error
	RetrieveAccessToken() (string, error)
	RetrieveRefreshToken() (string, error)
	Close() error
}
