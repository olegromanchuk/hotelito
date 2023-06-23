package secrets

type SecretsStore interface {
	StoreAccessToken(token string) error
	RetrieveAccessToken() (string, error)
}
