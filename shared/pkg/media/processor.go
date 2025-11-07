package media

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
)

// ImageSize represents dimensions for image resizing
type ImageSize struct {
	Width  int
	Height int
	Name   string // e.g., "small", "medium", "large"
}

// ProcessedImage represents a processed image variant
type ProcessedImage struct {
	Size       ImageSize
	Data       []byte
	FileSize   int64
	Format     string
}

// Processor handles image processing operations
type Processor struct {
	quality int
}

// NewProcessor creates a new image processor
func NewProcessor(quality int) *Processor {
	if quality <= 0 || quality > 100 {
		quality = 85 // default quality
	}
	return &Processor{
		quality: quality,
	}
}

// ResizeImage resizes an image to the specified dimensions
func (p *Processor) ResizeImage(src io.Reader, width, height int, format string) ([]byte, error) {
	// Decode the source image
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate aspect ratio preserving dimensions
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate target dimensions while preserving aspect ratio
	targetWidth, targetHeight := calculateDimensions(srcWidth, srcHeight, width, height)

	// Create target image
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Resize using high-quality algorithm
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// Encode to bytes
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: p.quality})
	case "png":
		err = png.Encode(&buf, dst)
	default:
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: p.quality})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateThumbnails generates multiple thumbnail sizes from an image
func (p *Processor) GenerateThumbnails(src io.Reader, sizes []ImageSize, format string) ([]ProcessedImage, error) {
	// Read all data into memory once
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	thumbnails := make([]ProcessedImage, 0, len(sizes))

	for _, size := range sizes {
		// Create a new reader for each size
		reader := bytes.NewReader(data)

		resized, err := p.ResizeImage(reader, size.Width, size.Height, format)
		if err != nil {
			return nil, fmt.Errorf("failed to resize to %s: %w", size.Name, err)
		}

		thumbnails = append(thumbnails, ProcessedImage{
			Size:     size,
			Data:     resized,
			FileSize: int64(len(resized)),
			Format:   format,
		})
	}

	return thumbnails, nil
}

// DetectImageFormat detects the format of an image
func DetectImageFormat(src io.Reader) (string, error) {
	_, format, err := image.DecodeConfig(src)
	if err != nil {
		return "", fmt.Errorf("failed to detect image format: %w", err)
	}
	return format, nil
}

// calculateDimensions calculates target dimensions while preserving aspect ratio
func calculateDimensions(srcWidth, srcHeight, maxWidth, maxHeight int) (int, int) {
	if srcWidth <= maxWidth && srcHeight <= maxHeight {
		return srcWidth, srcHeight
	}

	aspectRatio := float64(srcWidth) / float64(srcHeight)

	targetWidth := maxWidth
	targetHeight := int(float64(targetWidth) / aspectRatio)

	if targetHeight > maxHeight {
		targetHeight = maxHeight
		targetWidth = int(float64(targetHeight) * aspectRatio)
	}

	return targetWidth, targetHeight
}

// GetStandardProfilePictureSizes returns standard sizes for profile pictures
func GetStandardProfilePictureSizes() []ImageSize {
	return []ImageSize{
		{Width: 150, Height: 150, Name: "small"},
		{Width: 300, Height: 300, Name: "medium"},
		{Width: 600, Height: 600, Name: "large"},
	}
}

// GetStandardThumbnailSizes returns standard thumbnail sizes
func GetStandardThumbnailSizes() []ImageSize {
	return []ImageSize{
		{Width: 200, Height: 200, Name: "small"},
		{Width: 400, Height: 400, Name: "medium"},
		{Width: 800, Height: 800, Name: "large"},
	}
}
