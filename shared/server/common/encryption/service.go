package encryption

import "context"

type Service struct {
	manager *Manager
}

func NewService(cfg Config) (*Service, error) {
	mgr, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{manager: mgr}, nil
}

func (s *Service) Encrypt(ctx context.Context, plaintext []byte, opts EncryptOptions) (Ciphertext, error) {
	return s.manager.Encrypt(ctx, plaintext, opts)
}

func (s *Service) EncryptString(ctx context.Context, plaintext []byte, opts EncryptOptions) (string, error) {
	return s.manager.EncryptToString(ctx, plaintext, opts)
}

func (s *Service) Decrypt(ctx context.Context, ct Ciphertext, opts DecryptOptions) ([]byte, error) {
	return s.manager.Decrypt(ctx, ct, opts)
}

func (s *Service) DecryptString(ctx context.Context, encoded string, opts DecryptOptions) ([]byte, error) {
	return s.manager.DecryptString(ctx, encoded, opts)
}

func (s *Service) Rotate(key Key) error {
	return s.manager.Rotate(key)
}

func (s *Service) AddFallback(key Key) error {
	return s.manager.AddFallback(key)
}
