package request

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
)

const MaxBodySize = 1 << 20

type Parser struct {
	maxBodySize int64
}

func New() *Parser {
	return &Parser{
		maxBodySize: MaxBodySize,
	}
}

func (p *Parser) SetMaxBodySize(size int64) {
	p.maxBodySize = size
}

func (p *Parser) ParseJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	r.Body = http.MaxBytesReader(nil, r.Body, p.maxBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		return err
	}

	if decoder.More() {
		return fmt.Errorf("request body must contain only one JSON object")
	}

	return nil
}

func (p *Parser) ParseJSONAllowUnknown(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	r.Body = http.MaxBytesReader(nil, r.Body, p.maxBodySize)
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(v); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		return err
	}

	return nil
}

func (p *Parser) ParseQuery(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func (p *Parser) ParseQueryDefault(r *http.Request, key, defaultVal string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func (p *Parser) ParseQueryInt(r *http.Request, key string, defaultVal int) (int, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid integer value for %s", key)
	}

	return val, nil
}

func (p *Parser) ParseQueryInt64(r *http.Request, key string, defaultVal int64) (int64, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid integer value for %s", key)
	}

	return val, nil
}

func (p *Parser) ParseQueryFloat(r *http.Request, key string, defaultVal float64) (float64, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid float value for %s", key)
	}

	return val, nil
}

func (p *Parser) ParseQueryBool(r *http.Request, key string, defaultVal bool) (bool, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid boolean value for %s", key)
	}

	return val, nil
}

func (p *Parser) ParseQueryArray(r *http.Request, key string) []string {
	return r.URL.Query()[key]
}

func (p *Parser) ParseFormData(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form data: %w", err)
	}
	return nil
}

func (p *Parser) GetFormValue(r *http.Request, key string) string {
	return r.FormValue(key)
}

func (p *Parser) GetFormValueDefault(r *http.Request, key, defaultVal string) string {
	val := r.FormValue(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func (p *Parser) ParseMultipartForm(r *http.Request, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}
	return nil
}

func (p *Parser) GetPathParam(r *http.Request, key string) string {
	return r.PathValue(key)
}

const DefaultMaxBodySize = 1 << 20

type ValidationErrorDetail struct {
	Msg  string
	Code string
}

const (
	REQUIRED_FIELD   = "REQUIRED_FIELD"
	INVALID_FORMAT   = "INVALID_FORMAT"
	TOO_SHORT        = "TOO_SHORT"
	TOO_LONG         = "TOO_LONG"
	PATTERN_MISMATCH = "PATTERN_MISMATCH"
)

type Validator interface {
	GetValue() interface{}
	ValidateErrors(validator.ValidationErrors) ([]ValidationErrorDetail, error)
}

type Config struct {
	MaxBodySize        int64
	DisallowUnknown    bool
	RequireContentType bool
	AllowEmptyBody     bool
}
