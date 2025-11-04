package hashing

import "context"

type HashingService struct {
	manager *Manager
}

func NewService(cfg Config) (*HashingService, error) {
	mgr, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}
	service := &HashingService{manager: mgr}
	service.manager.cfg.validate()
	return service, nil
}

func (s *HashingService) DefaultAlgorithm() Algorithm {
	return s.manager.cfg.Default
}

func (s *HashingService) HashPassword(ctx context.Context, password string) (*HashResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.manager.Hash(password)
}

func (s *HashingService) HashPasswordWithAlgorithm(ctx context.Context, algo Algorithm, password string) (*HashResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.manager.HashWithAlgorithm(algo, password)
}

func (s *HashingService) VerifyPassword(ctx context.Context, password string, encoded string) (bool, Algorithm, error) {
	if err := ctx.Err(); err != nil {
		return false, "", err
	}
	return s.manager.Verify(password, encoded)
}

func (s *HashingService) NeedsRehash(encoded string) (bool, error) {
	return s.manager.NeedsRehash(encoded)
}

func (s *HashingService) VerifyAndRehash(ctx context.Context, password, encoded string) (bool, *HashResult, error) {
	ok, algo, err := s.VerifyPassword(ctx, password, encoded)
	if err != nil || !ok {
		return ok, nil, err
	}
	rehash, err := s.NeedsRehash(encoded)
	if err != nil {
		return ok, nil, err
	}
	if !rehash {
		return ok, nil, nil
	}
	res, err := s.HashPassword(ctx, password)
	if err != nil {
		return ok, nil, err
	}
	if res != nil {
		res.Params["previous"] = string(algo)
	}
	return ok, res, nil
}
