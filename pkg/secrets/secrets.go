package secrets

type SecretsStore interface {
	StoreAccessToken(token string) error
	StoreRefreshToken(token string) error
	RetrieveAccessToken() (string, error)
	RetrieveRefreshToken() (string, error)
	StoreOauthState(state string) error
	RetrieveOauthState(state string) (string, error)
	RetrieveVar(varName string) (varValue string, err error)
	Close() error
}
