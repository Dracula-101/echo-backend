package request

import (
	"fmt"
	"mime/multipart"
	"strconv"
)

const (
	DefaultMaxMemory = 32 << 20 // 32 MB
)

// FormValue extracts a form value from the request
func (h *RequestHandler) FormValue(key string) string {
	return h.request.FormValue(key)
}

// FormValueDefault extracts a form value with a default value
func (h *RequestHandler) FormValueDefault(key, defaultVal string) string {
	val := h.request.FormValue(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// FormValueInt extracts an integer form value
func (h *RequestHandler) FormValueInt(key string, defaultVal int) (int, error) {
	str := h.request.FormValue(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid integer value for %s", key)
	}

	return val, nil
}

// FormValueInt64 extracts an int64 form value
func (h *RequestHandler) FormValueInt64(key string, defaultVal int64) (int64, error) {
	str := h.request.FormValue(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid integer value for %s", key)
	}

	return val, nil
}

// FormValueFloat extracts a float64 form value
func (h *RequestHandler) FormValueFloat(key string, defaultVal float64) (float64, error) {
	str := h.request.FormValue(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid float value for %s", key)
	}

	return val, nil
}

// FormValueBool extracts a boolean form value
func (h *RequestHandler) FormValueBool(key string, defaultVal bool) (bool, error) {
	str := h.request.FormValue(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid boolean value for %s", key)
	}

	return val, nil
}

// FormValueArray extracts an array of form values
func (h *RequestHandler) FormValueArray(key string) []string {
	if h.request.Form == nil {
		h.request.ParseForm()
	}
	return h.request.Form[key]
}

// PostFormValue extracts a POST form value (not from URL query)
func (h *RequestHandler) PostFormValue(key string) string {
	return h.request.PostFormValue(key)
}

// PostFormValueDefault extracts a POST form value with a default value
func (h *RequestHandler) PostFormValueDefault(key, defaultVal string) string {
	val := h.request.PostFormValue(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// PostFormValueArray extracts an array of POST form values
func (h *RequestHandler) PostFormValueArray(key string) []string {
	if h.request.PostForm == nil {
		h.request.ParseForm()
	}
	return h.request.PostForm[key]
}

// ParseForm parses the form data from the request
func (h *RequestHandler) ParseForm() error {
	if err := h.request.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %v", err)
	}
	return nil
}

// ParseMultipartForm parses the multipart form data from the request
func (h *RequestHandler) ParseMultipartForm(maxMemory int64) error {
	if err := h.request.ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("failed to parse multipart form: %v", err)
	}
	return nil
}

// GetFormFile extracts a single file from multipart form
func (h *RequestHandler) GetFormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	file, fileHeader, err := h.request.FormFile(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get form file %s: %v", key, err)
	}
	return file, fileHeader, nil
}

// GetFormFiles extracts multiple files from multipart form
func (h *RequestHandler) GetFormFiles(key string) ([]*multipart.FileHeader, error) {
	if h.request.MultipartForm == nil {
		if err := h.ParseMultipartForm(DefaultMaxMemory); err != nil {
			return nil, err
		}
	}

	files, ok := h.request.MultipartForm.File[key]
	if !ok {
		return nil, fmt.Errorf("no files found for key %s", key)
	}

	return files, nil
}

// GetAllFormFiles extracts all files from multipart form
func (h *RequestHandler) GetAllFormFiles() (map[string][]*multipart.FileHeader, error) {
	if h.request.MultipartForm == nil {
		if err := h.ParseMultipartForm(DefaultMaxMemory); err != nil {
			return nil, err
		}
	}

	return h.request.MultipartForm.File, nil
}

// HasFormValue checks if a form value exists
func (h *RequestHandler) HasFormValue(key string) bool {
	if h.request.Form == nil {
		h.ParseForm()
	}
	_, ok := h.request.Form[key]
	return ok
}

// HasFormFile checks if a form file exists
func (h *RequestHandler) HasFormFile(key string) bool {
	if h.request.MultipartForm == nil {
		h.ParseMultipartForm(DefaultMaxMemory)
	}
	if h.request.MultipartForm == nil {
		return false
	}
	_, ok := h.request.MultipartForm.File[key]
	return ok
}
