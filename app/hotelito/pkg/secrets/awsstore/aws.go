package secrets

type AWSSecretsStore struct {
	// ...
}

func (s *AWSSecretsStore) StoreAccessToken(token string) error {
	// implementation using AWS Secrets
}

func (s *AWSSecretsStore) RetrieveAccessToken() (string, error) {
	// implementation using AWS Secrets
}
