package media

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/webp"
)

type MediaInfo struct {
	// File basics
	FileName      string
	FileType      string
	MimeType      string
	FileCategory  string
	FileExtension string
	FileSizeBytes int64

	// Image/Video dimensions
	Width  *int
	Height *int

	// Video/Audio specific
	DurationSeconds *float64
	Bitrate         *int
	FrameRate       *float64
	Codec           *string
	Resolution      *string
	AspectRatio     *string
	VideoCodec      *string
	AudioCodec      *string
	SubtitleTracks  *int
	AudioChannels   *int
	SampleRate      *int

	// Image specific
	ColorProfile    *string
	Orientation     *string
	HasAlphaChannel *bool
	DominantColors  []string

	// Document specific
	PageCount *int
	WordCount *int

	// EXIF data
	ExifData     map[string]interface{}
	GPSLatitude  *float64
	GPSLongitude *float64
	GPSAltitude  *float64
	CameraMake   *string
	CameraModel  *string
	LensModel    *string
	FocalLength  *float64
	Aperture     *float64
	ISO          *int
	ShutterSpeed *string
	CaptureDate  *time.Time
}

// Processor handles media file processing
type Processor struct {
	imageProcessor *ImageProcessor
	videoProcessor *VideoProcessor
}

// NewProcessor creates a new media processor
func NewProcessor() *Processor {
	return &Processor{
		imageProcessor: NewImageProcessor(),
		videoProcessor: NewVideoProcessor(),
	}
}

// Process analyzes a media file and extracts metadata
func (p *Processor) Process(ctx context.Context, filePath string) (*MediaInfo, error) {
	info := &MediaInfo{
		FileName:      filepath.Base(filePath),
		FileExtension: strings.TrimPrefix(filepath.Ext(filePath), "."),
	}

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	info.FileSizeBytes = fileInfo.Size()

	// Detect MIME type
	mimeType, err := detectMimeType(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect mime type: %w", err)
	}
	info.MimeType = mimeType
	info.FileType = mimeType

	// Determine category and process accordingly
	category := categorizeFile(mimeType, info.FileExtension)
	info.FileCategory = category

	switch category {
	case "image":
		if err := p.imageProcessor.Process(ctx, filePath, info); err != nil {
			return nil, fmt.Errorf("failed to process image: %w", err)
		}
	case "video":
		if err := p.videoProcessor.Process(ctx, filePath, info); err != nil {
			return nil, fmt.Errorf("failed to process video: %w", err)
		}
	default:
		// Generic file, no special processing
	}

	return info, nil
}

func detectMimeType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	mimeType := detectMimeFromContent(buffer[:n])
	if mimeType != "" {
		return mimeType, nil
	}
	ext := filepath.Ext(filePath)
	mimeType = mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return mimeType, nil
}

func detectMimeFromContent(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	switch {
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a":
		return "image/gif"
	case len(data) >= 12 && string(data[8:12]) == "WEBP":
		return "image/webp"
	case len(data) >= 4 && string(data[:4]) == "ftyp":
		return "video/mp4"
	case len(data) >= 4 && (string(data[:4]) == "RIFF"):
		if len(data) >= 12 && string(data[8:12]) == "WEBP" {
			return "image/webp"
		}
		return "video/avi"
	case len(data) >= 3 && string(data[:3]) == "ID3":
		return "audio/mpeg"
	case len(data) >= 4 && string(data[:4]) == "%PDF":
		return "application/pdf"
	}

	return ""
}

func categorizeFile(mimeType, extension string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}

	ext := strings.ToLower(extension)
	switch ext {
	case "pdf", "doc", "docx", "txt", "rtf", "odt":
		return "document"
	case "xls", "xlsx", "csv", "ods":
		return "document"
	case "ppt", "pptx", "odp":
		return "document"
	case "mp4", "mov", "avi", "mkv", "webm", "flv", "wmv":
		return "video"
	case "mp3", "wav", "flac", "aac", "ogg", "m4a", "wma":
		return "audio"
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "tiff":
		return "image"
	default:
		return "other"
	}
}

func CalculateAspectRatio(width, height int) string {
	if height == 0 {
		return ""
	}

	ratio := float64(width) / float64(height)

	switch {
	case ratio > 1.77 && ratio < 1.78:
		return "16:9"
	case ratio > 1.33 && ratio < 1.34:
		return "4:3"
	case ratio > 1.59 && ratio < 1.61:
		return "16:10"
	case ratio > 2.33 && ratio < 2.40:
		return "21:9"
	case ratio > 0.99 && ratio < 1.01:
		return "1:1"
	default:
		return fmt.Sprintf("%.2f:1", ratio)
	}
}

func ProcessExifData(filePath string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		return nil, nil
	}

	exifData := make(map[string]interface{})

	if make, err := x.Get(exif.Make); err == nil {
		exifData["Make"], _ = make.StringVal()
	}
	if model, err := x.Get(exif.Model); err == nil {
		exifData["Model"], _ = model.StringVal()
	}
	if orientation, err := x.Get(exif.Orientation); err == nil {
		exifData["Orientation"], _ = orientation.Int(0)
	}
	if dateTime, err := x.Get(exif.DateTime); err == nil {
		exifData["DateTime"], _ = dateTime.StringVal()
	}

	return exifData, nil
}

func ExtractGPSData(filePath string) (lat, lon, alt *float64, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, nil, err
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		return nil, nil, nil, nil // No EXIF data
	}

	latitude, longitude, err := x.LatLong()
	if err == nil {
		lat = &latitude
		lon = &longitude
	}

	return lat, lon, nil, nil
}
