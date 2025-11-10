package middleware

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"shared/pkg/logger"
	"shared/server/response"
)

func FileOnlyMultipart(log logger.Logger, maxBodySize int64, allowedTypes []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				next.ServeHTTP(w, r)
				return
			}

			// Check Content-Length header before parsing to avoid unexpected EOF
			if r.ContentLength > maxBodySize {
				log.Error("Request body size exceeds limit",
					logger.Int64("content_length", r.ContentLength),
					logger.Int64("max_body_size", maxBodySize),
				)
				response.BadRequestError(ctx, r, w,
					fmt.Sprintf("Request body size (%d bytes) exceeds maximum allowed size",
						r.ContentLength),
					fmt.Errorf("body size limit exceeded - (max %d bytes)", maxBodySize))
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

			maxMemory := maxBodySize / 5
			if maxMemory < 10<<20 {
				maxMemory = 10 << 20 // 10MB minimum
			}
			if maxMemory > 100<<20 {
				maxMemory = 100 << 20 // 100MB maximum
			}

			if err := r.ParseMultipartForm(maxMemory); err != nil {
				log.Error("Failed to parse multipart form",
					logger.Error(err),
					logger.Int64("max_memory", maxMemory),
					logger.Int64("max_body_size", maxBodySize),
					logger.Int64("content_length", r.ContentLength),
				)
				response.BadRequestError(ctx, r, w, "Failed to parse multipart form. The request may be too large or malformed.", err)
				return
			}

			if r.MultipartForm == nil {
				log.Error("Multipart form is nil after parsing")
				response.BadRequestError(ctx, r, w, "Invalid multipart form data", fmt.Errorf("multipart form is nil"))
				return
			}

			// Check for non-file fields
			if len(r.MultipartForm.Value) > 0 {
				log.Error("Non-file fields detected in multipart form",
					logger.Int("text_fields_count", len(r.MultipartForm.Value)),
				)
				response.BadRequestError(ctx, r, w, "Only file uploads are allowed, no text form fields permitted", fmt.Errorf("non-file fields found"))
				return
			}

			// Check if files exist
			if len(r.MultipartForm.File) == 0 {
				log.Error("No files in multipart form")
				response.BadRequestError(ctx, r, w, "At least one file is required", fmt.Errorf("no files found"))
				return
			}

			// Validate total size and individual files
			var totalSize int64
			for fieldName, files := range r.MultipartForm.File {
				for idx, fileHeader := range files {
					totalSize += fileHeader.Size

					// Check file type
					if len(allowedTypes) > 0 {
						contentType := fileHeader.Header.Get("Content-Type")
						if !isAllowedType(contentType, allowedTypes) {
							log.Error("File type not allowed",
								logger.String("field_name", fieldName),
								logger.Int("file_index", idx),
								logger.String("file_name", fileHeader.Filename),
								logger.String("content_type", contentType),
								logger.Any("allowed_types", allowedTypes),
							)
							response.BadRequestError(ctx, r, w,
								fmt.Sprintf("File type '%s' is not allowed.",
									contentType),
								fmt.Errorf("file type not allowed"))
							return
						}
					}

					// Validate file integrity by attempting to open it
					file, err := fileHeader.Open()
					if err != nil {
						log.Error("Failed to open uploaded file (possibly corrupt)",
							logger.String("field_name", fieldName),
							logger.Int("file_index", idx),
							logger.String("file_name", fileHeader.Filename),
							logger.Error(err),
						)
						response.BadRequestError(ctx, r, w,
							fmt.Sprintf("File '%s' appears to be corrupt or cannot be read", fileHeader.Filename),
							err)
						return
					}

					// Try to read a small portion to verify the file is readable
					buf := make([]byte, 512) // Read first 512 bytes
					_, err = file.Read(buf)
					if err != nil && err != io.EOF {
						file.Close()
						log.Error("Failed to read uploaded file (possibly corrupt)",
							logger.String("field_name", fieldName),
							logger.Int("file_index", idx),
							logger.String("file_name", fileHeader.Filename),
							logger.Error(err),
						)
						response.BadRequestError(ctx, r, w,
							fmt.Sprintf("File '%s' appears to be corrupt or cannot be read", fileHeader.Filename),
							err)
						return
					}

					// Close the file and reset for handler use
					file.Close()
				}
			}

			// Check total size against limit
			if totalSize > maxBodySize {
				log.Error("Total file size exceeds limit",
					logger.Int64("total_size", totalSize),
					logger.Int64("max_size", maxBodySize),
				)
				response.BadRequestError(ctx, r, w,
					fmt.Sprintf("Total file size (%d bytes) exceeds maximum allowed size (%d bytes)",
						totalSize, maxBodySize),
					fmt.Errorf("size limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isAllowedType checks if the given content type is in the allowed list
func isAllowedType(contentType string, allowedTypes []string) bool {
	// Normalize content type (remove parameters like charset)
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	for _, allowed := range allowedTypes {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if contentType == allowed {
			return true
		}

		// Support wildcard matching (e.g., "image/*")
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(contentType, prefix+"/") {
				return true
			}
		}
	}

	return false
}
