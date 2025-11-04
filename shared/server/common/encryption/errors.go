package encryption

import "errors"

var (
	ErrInvalidCiphertext = errors.New("encryption: invalid ciphertext")
	ErrInvalidCipher     = errors.New("encryption: invalid cipher")
	ErrKeyNotFound       = errors.New("encryption: key not found")
	ErrInvalidConfig     = errors.New("encryption: invalid config")
)
