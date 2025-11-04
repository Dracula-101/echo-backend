package hashing

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

type Algorithm string

type Manager struct {
	cfg Config
}

type Config struct {
	Default Algorithm
	Argon2  Argon2Config
	Bcrypt  BcryptConfig
	Scrypt  ScryptConfig
}

type Argon2Config struct {
	Time       uint32
	Memory     uint32
	Threads    uint8
	KeyLength  uint32
	SaltLength uint32
}

type BcryptConfig struct {
	Cost int
}

type ScryptConfig struct {
	N          int
	R          int
	P          int
	KeyLength  int
	SaltLength int
}

type HashResult struct {
	Algorithm Algorithm
	Encoded   string
	Salt      []byte
	Sum       []byte
	Params    map[string]string
}

func (r *HashResult) String() string {
	if r == nil {
		return ""
	}
	return r.Encoded
}

type parsedHash struct {
	algorithm Algorithm
	params    map[string]string
	salt      []byte
	sum       []byte
}

const (
	AlgorithmArgon2id Algorithm = "argon2id"
	AlgorithmBcrypt   Algorithm = "bcrypt"
	AlgorithmScrypt   Algorithm = "scrypt"
)

func NewManager(cfg Config) (*Manager, error) {
	cfg = applyDefaults(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg}, nil
}

func applyDefaults(cfg Config) Config {
	if cfg.Default == "" {
		cfg.Default = AlgorithmArgon2id
	}
	if cfg.Argon2.Time == 0 {
		cfg.Argon2.Time = 3
	}
	if cfg.Argon2.Memory == 0 {
		cfg.Argon2.Memory = 64 * 1024
	}
	if cfg.Argon2.Threads == 0 {
		cfg.Argon2.Threads = 2
	}
	if cfg.Argon2.KeyLength == 0 {
		cfg.Argon2.KeyLength = 32
	}
	if cfg.Argon2.SaltLength == 0 {
		cfg.Argon2.SaltLength = 16
	}
	if cfg.Bcrypt.Cost == 0 {
		cfg.Bcrypt.Cost = bcrypt.DefaultCost
	}
	if cfg.Scrypt.N == 0 {
		cfg.Scrypt.N = 1 << 15
	}
	if cfg.Scrypt.R == 0 {
		cfg.Scrypt.R = 8
	}
	if cfg.Scrypt.P == 0 {
		cfg.Scrypt.P = 1
	}
	if cfg.Scrypt.KeyLength == 0 {
		cfg.Scrypt.KeyLength = 32
	}
	if cfg.Scrypt.SaltLength == 0 {
		cfg.Scrypt.SaltLength = 16
	}
	return cfg
}

func (c Config) validate() error {
	switch c.Default {
	case AlgorithmArgon2id, AlgorithmBcrypt, AlgorithmScrypt:
	default:
		return fmt.Errorf("%w: %s", ErrUnknownAlgorithm, c.Default)
	}
	if c.Argon2.Time < 1 {
		return fmt.Errorf("%w: argon2 time must be at least 1", ErrInvalidConfig)
	}
	if c.Argon2.Memory < 8*1024 {
		return fmt.Errorf("%w: argon2 memory must be at least 8192 KiB", ErrInvalidConfig)
	}
	if c.Argon2.Threads < 1 {
		return fmt.Errorf("%w: argon2 threads must be at least 1", ErrInvalidConfig)
	}
	if c.Argon2.SaltLength < 8 {
		return fmt.Errorf("%w: argon2 salt length must be at least 8", ErrInvalidConfig)
	}
	if c.Argon2.KeyLength < 16 {
		return fmt.Errorf("%w: argon2 key length must be at least 16", ErrInvalidConfig)
	}
	if c.Bcrypt.Cost < bcrypt.MinCost || c.Bcrypt.Cost > bcrypt.MaxCost {
		return fmt.Errorf("%w: bcrypt cost out of range", ErrInvalidConfig)
	}
	if c.Scrypt.N <= 1 || c.Scrypt.N&(c.Scrypt.N-1) != 0 {
		return fmt.Errorf("%w: scrypt N must be a power of 2 greater than 1", ErrInvalidConfig)
	}
	if c.Scrypt.R <= 0 {
		return fmt.Errorf("%w: scrypt R must be greater than 0", ErrInvalidConfig)
	}
	if c.Scrypt.P <= 0 {
		return fmt.Errorf("%w: scrypt P must be greater than 0", ErrInvalidConfig)
	}
	if c.Scrypt.KeyLength < 16 {
		return fmt.Errorf("%w: scrypt key length must be at least 16", ErrInvalidConfig)
	}
	if c.Scrypt.SaltLength < 8 {
		return fmt.Errorf("%w: scrypt salt length must be at least 8", ErrInvalidConfig)
	}
	return nil
}

func (m *Manager) Hash(password string) (*HashResult, error) {
	return m.HashWithAlgorithm(m.cfg.Default, password)
}

func (m *Manager) HashWithAlgorithm(algo Algorithm, password string) (*HashResult, error) {
	switch algo {
	case AlgorithmArgon2id:
		return m.hashArgon(password)
	case AlgorithmBcrypt:
		return m.hashBcrypt(password)
	case AlgorithmScrypt:
		return m.hashScrypt(password)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, algo)
	}
}

func (m *Manager) Verify(password string, encoded string) (bool, Algorithm, error) {
	parsed, err := parseEncoded(encoded)
	if err != nil {
		return false, "", err
	}
	switch parsed.algorithm {
	case AlgorithmArgon2id:
		return verifyArgon(password, parsed)
	case AlgorithmBcrypt:
		return verifyBcrypt(password, parsed)
	case AlgorithmScrypt:
		return verifyScrypt(password, parsed)
	default:
		return false, parsed.algorithm, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, parsed.algorithm)
	}
}

func (m *Manager) NeedsRehash(encoded string) (bool, error) {
	parsed, err := parseEncoded(encoded)
	if err != nil {
		return false, err
	}
	switch parsed.algorithm {
	case AlgorithmArgon2id:
		return needsRehashArgon(parsed, m.cfg.Argon2), nil
	case AlgorithmBcrypt:
		return needsRehashBcrypt(parsed, m.cfg.Bcrypt), nil
	case AlgorithmScrypt:
		return needsRehashScrypt(parsed, m.cfg.Scrypt), nil
	default:
		return false, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, parsed.algorithm)
	}
}

func (m *Manager) hashArgon(password string) (*HashResult, error) {
	salt := make([]byte, m.cfg.Argon2.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	sum := argon2.IDKey([]byte(password), salt, m.cfg.Argon2.Time, m.cfg.Argon2.Memory, m.cfg.Argon2.Threads, m.cfg.Argon2.KeyLength)
	params := map[string]string{
		"v": "19",
		"m": strconv.FormatUint(uint64(m.cfg.Argon2.Memory), 10),
		"t": strconv.FormatUint(uint64(m.cfg.Argon2.Time), 10),
		"p": strconv.FormatUint(uint64(m.cfg.Argon2.Threads), 10),
	}
	encoded := fmt.Sprintf("$%s$v=%s$m=%s,t=%s,p=%s$%s$%s",
		AlgorithmArgon2id,
		params["v"],
		params["m"],
		params["t"],
		params["p"],
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	)
	return &HashResult{
		Algorithm: AlgorithmArgon2id,
		Encoded:   encoded,
		Salt:      salt,
		Sum:       sum,
		Params:    params,
	}, nil
}

func (m *Manager) hashBcrypt(password string) (*HashResult, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), m.cfg.Bcrypt.Cost)
	if err != nil {
		return nil, err
	}

	var salt []byte
	if len(hash) >= 29 {
		saltEncoded := string(hash[7:29])
		salt, _ = bcrypt64Decode(saltEncoded)
	}

	params := map[string]string{
		"c": strconv.Itoa(m.cfg.Bcrypt.Cost),
	}
	encoded := fmt.Sprintf("$%s$c=%s$%s",
		AlgorithmBcrypt,
		params["c"],
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return &HashResult{
		Algorithm: AlgorithmBcrypt,
		Encoded:   encoded,
		Salt:      salt,
		Sum:       hash,
		Params:    params,
	}, nil
}

func (m *Manager) hashScrypt(password string) (*HashResult, error) {
	salt := make([]byte, m.cfg.Scrypt.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	sum, err := scrypt.Key([]byte(password), salt, m.cfg.Scrypt.N, m.cfg.Scrypt.R, m.cfg.Scrypt.P, m.cfg.Scrypt.KeyLength)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"n": strconv.Itoa(m.cfg.Scrypt.N),
		"r": strconv.Itoa(m.cfg.Scrypt.R),
		"p": strconv.Itoa(m.cfg.Scrypt.P),
	}
	encoded := fmt.Sprintf("$%s$n=%s,r=%s,p=%s$%s$%s",
		AlgorithmScrypt,
		params["n"],
		params["r"],
		params["p"],
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	)
	return &HashResult{
		Algorithm: AlgorithmScrypt,
		Encoded:   encoded,
		Salt:      salt,
		Sum:       sum,
		Params:    params,
	}, nil
}

func verifyArgon(password string, parsed *parsedHash) (bool, Algorithm, error) {
	timeParam, err := parseUint(parsed.params["t"], 10)
	if err != nil {
		return false, parsed.algorithm, err
	}
	memoryParam, err := parseUint(parsed.params["m"], 10)
	if err != nil {
		return false, parsed.algorithm, err
	}
	threadsParam, err := parseUint(parsed.params["p"], 10)
	if err != nil {
		return false, parsed.algorithm, err
	}
	sum := argon2.IDKey([]byte(password), parsed.salt, uint32(timeParam), uint32(memoryParam), uint8(threadsParam), uint32(len(parsed.sum)))
	return subtle.ConstantTimeCompare(sum, parsed.sum) == 1, parsed.algorithm, nil
}

func verifyBcrypt(password string, parsed *parsedHash) (bool, Algorithm, error) {
	if len(parsed.sum) == 0 {
		return false, parsed.algorithm, ErrMalformedHash
	}
	err := bcrypt.CompareHashAndPassword(parsed.sum, []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, parsed.algorithm, nil
	}
	return err == nil, parsed.algorithm, err
}

func verifyScrypt(password string, parsed *parsedHash) (bool, Algorithm, error) {
	nVar, err := strconv.Atoi(parsed.params["n"])
	if err != nil {
		return false, parsed.algorithm, err
	}
	rVar, err := strconv.Atoi(parsed.params["r"])
	if err != nil {
		return false, parsed.algorithm, err
	}
	pVar, err := strconv.Atoi(parsed.params["p"])
	if err != nil {
		return false, parsed.algorithm, err
	}
	sum, err := scrypt.Key([]byte(password), parsed.salt, nVar, rVar, pVar, len(parsed.sum))
	if err != nil {
		return false, parsed.algorithm, err
	}
	return subtle.ConstantTimeCompare(sum, parsed.sum) == 1, parsed.algorithm, nil
}

func needsRehashArgon(parsed *parsedHash, cfg Argon2Config) bool {
	return parsed.params["m"] != strconv.FormatUint(uint64(cfg.Memory), 10) ||
		parsed.params["t"] != strconv.FormatUint(uint64(cfg.Time), 10) ||
		parsed.params["p"] != strconv.FormatUint(uint64(cfg.Threads), 10) ||
		len(parsed.sum) != int(cfg.KeyLength)
}

func needsRehashBcrypt(parsed *parsedHash, cfg BcryptConfig) bool {
	return parsed.params["c"] != strconv.Itoa(cfg.Cost)
}

func needsRehashScrypt(parsed *parsedHash, cfg ScryptConfig) bool {
	return parsed.params["n"] != strconv.Itoa(cfg.N) ||
		parsed.params["r"] != strconv.Itoa(cfg.R) ||
		parsed.params["p"] != strconv.Itoa(cfg.P) ||
		len(parsed.sum) != cfg.KeyLength
}

func parseEncoded(encoded string) (*parsedHash, error) {
	if encoded == "" {
		return nil, ErrMalformedHash
	}
	if !strings.HasPrefix(encoded, "$") {
		return nil, fmt.Errorf("%w: must start with $, but got: %s", ErrMalformedHash, encoded)
	}
	parts := strings.Split(encoded[1:], "$")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: insufficient parts", ErrMalformedHash)
	}
	algo := Algorithm(parts[0])
	params := map[string]string{}
	var paramSegment, saltSegment, hashSegment string
	switch algo {
	case AlgorithmArgon2id, AlgorithmScrypt:
		if len(parts) == 4 {
			paramSegment = parts[1]
			saltSegment = parts[2]
			hashSegment = parts[3]
		} else if len(parts) == 5 && strings.HasPrefix(parts[1], "v=") {
			paramSegment = parts[2]
			saltSegment = parts[3]
			hashSegment = parts[4]
		} else {
			return nil, fmt.Errorf("%w: expected 4 or 5 parts for %s, got %d", ErrMalformedHash, algo, len(parts))
		}
	case AlgorithmBcrypt:
		if len(parts) < 2 {
			return nil, fmt.Errorf("%w: expected at least 2 parts for bcrypt, got %d", ErrMalformedHash, len(parts))
		}
		hashSegment = strings.Join(parts[1:], "$")
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownAlgorithm, string(algo))
	}
	if paramSegment != "" {
		for _, kv := range strings.Split(paramSegment, ",") {
			if kv == "" {
				continue
			}
			pair := strings.SplitN(kv, "=", 2)
			if len(pair) == 2 {
				params[pair[0]] = pair[1]
			}
		}
	}
	decodeBase64 := func(s string) ([]byte, error) {
		if s == "" {
			return nil, nil
		}
		if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
			return b, nil
		}
		if b, err := base64.StdEncoding.DecodeString(s); err == nil {
			return b, nil
		}
		return nil, errors.New("invalid base64")
	}
	var salt []byte
	var err error
	if saltSegment != "" {
		salt, err = decodeBase64(saltSegment)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid salt encoding", ErrMalformedHash)
		}
	}
	var sum []byte
	if algo == AlgorithmBcrypt {
		sum = []byte(hashSegment)
	} else {
		sum, err = decodeBase64(hashSegment)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid hash encoding", ErrMalformedHash)
		}
	}
	return &parsedHash{
		algorithm: algo,
		params:    params,
		salt:      salt,
		sum:       sum,
	}, nil
}

func parseUint(value string, base int) (uint64, error) {
	if value == "" {
		return 0, ErrMalformedHash
	}
	parsed, err := strconv.ParseUint(value, base, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func bcrypt64Decode(src string) ([]byte, error) {
	const bcryptAlphabet = "./ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const stdAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	translated := make([]byte, len(src))
	for i := 0; i < len(src); i++ {
		idx := strings.IndexByte(bcryptAlphabet, src[i])
		if idx == -1 {
			return nil, fmt.Errorf("invalid bcrypt base64 character")
		}
		translated[i] = stdAlphabet[idx]
	}

	return base64.RawStdEncoding.DecodeString(string(translated))
}
