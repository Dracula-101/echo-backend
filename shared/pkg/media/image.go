package media

import (
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/webp"
)

// ImageProcessor handles image file processing
type ImageProcessor struct{}

// NewImageProcessor creates a new image processor
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

// Process extracts metadata from an image file
func (p *ImageProcessor) Process(ctx context.Context, filePath string, info *MediaInfo) error {
	// Open and decode image
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Extract dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	info.Width = &width
	info.Height = &height

	// Calculate aspect ratio
	aspectRatio := CalculateAspectRatio(width, height)
	info.AspectRatio = &aspectRatio

	// Determine if image has alpha channel
	hasAlpha := hasAlphaChannel(img)
	info.HasAlphaChannel = &hasAlpha

	// Get color profile (simplified)
	colorProfile := getColorProfile(format, img)
	info.ColorProfile = &colorProfile

	// Process EXIF data
	if err := p.processExifData(filePath, info); err != nil {
		// EXIF processing is optional, log but don't fail
		fmt.Printf("Warning: failed to process EXIF data: %v\n", err)
	}

	// Extract dominant colors (basic implementation)
	colors := p.extractDominantColors(img)
	info.DominantColors = colors

	return nil
}

// processExifData extracts EXIF metadata from image
func (p *ImageProcessor) processExifData(filePath string, info *MediaInfo) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		// No EXIF data available
		return nil
	}

	info.ExifData = make(map[string]interface{})

	// Camera Make
	if make, err := x.Get(exif.Make); err == nil {
		if val, err := make.StringVal(); err == nil {
			info.CameraMake = &val
			info.ExifData["Make"] = val
		}
	}

	// Camera Model
	if model, err := x.Get(exif.Model); err == nil {
		if val, err := model.StringVal(); err == nil {
			info.CameraModel = &val
			info.ExifData["Model"] = val
		}
	}

	// Orientation
	if orientation, err := x.Get(exif.Orientation); err == nil {
		if val, err := orientation.Int(0); err == nil {
			orientationStr := getOrientationString(val)
			info.Orientation = &orientationStr
			info.ExifData["Orientation"] = val
		}
	}

	// Focal Length
	if focalLength, err := x.Get(exif.FocalLength); err == nil {
		if num, denom, err := focalLength.Rat2(0); err == nil && denom != 0 {
			fl := float64(num) / float64(denom)
			info.FocalLength = &fl
			info.ExifData["FocalLength"] = fl
		}
	}

	// F-Number (Aperture)
	if fNumber, err := x.Get(exif.FNumber); err == nil {
		if num, denom, err := fNumber.Rat2(0); err == nil && denom != 0 {
			aperture := float64(num) / float64(denom)
			info.Aperture = &aperture
			info.ExifData["Aperture"] = aperture
		}
	}

	// ISO
	if isoTag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if iso, err := isoTag.Int(0); err == nil {
			info.ISO = &iso
			info.ExifData["ISO"] = iso
		}
	}

	// Exposure Time (Shutter Speed)
	if exposureTime, err := x.Get(exif.ExposureTime); err == nil {
		if num, denom, err := exposureTime.Rat2(0); err == nil {
			shutterSpeed := fmt.Sprintf("%d/%d", num, denom)
			info.ShutterSpeed = &shutterSpeed
			info.ExifData["ShutterSpeed"] = shutterSpeed
		}
	}

	// DateTime
	if dateTime, err := x.Get(exif.DateTime); err == nil {
		if dtStr, err := dateTime.StringVal(); err == nil {
			info.ExifData["DateTime"] = dtStr
			// Parse datetime if needed
			// captureDate, _ := time.Parse("2006:01:02 15:04:05", dtStr)
			// info.CaptureDate = &captureDate
		}
	}

	// GPS Data
	lat, lon, err := x.LatLong()
	if err == nil {
		info.GPSLatitude = &lat
		info.GPSLongitude = &lon
		info.ExifData["GPSLatitude"] = lat
		info.ExifData["GPSLongitude"] = lon
	}

	return nil
}

// hasAlphaChannel checks if an image has an alpha channel
func hasAlphaChannel(img image.Image) bool {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA, *image.RGBA64, *image.NRGBA64:
		return true
	default:
		return false
	}
}

// getColorProfile returns a simplified color profile description
func getColorProfile(format string, img image.Image) string {
	switch img.ColorModel() {
	case color.GrayModel:
		return "grayscale"
	case color.Gray16Model:
		return "grayscale-16bit"
	case color.RGBAModel, color.NRGBAModel:
		return "rgba"
	case color.RGBA64Model, color.NRGBA64Model:
		return "rgba-64bit"
	default:
		return "rgb"
	}
}

// getOrientationString converts EXIF orientation number to string
func getOrientationString(orientation int) string {
	switch orientation {
	case 1:
		return "normal"
	case 2:
		return "flip-horizontal"
	case 3:
		return "rotate-180"
	case 4:
		return "flip-vertical"
	case 5:
		return "transpose"
	case 6:
		return "rotate-90"
	case 7:
		return "transverse"
	case 8:
		return "rotate-270"
	default:
		return "unknown"
	}
}

// extractDominantColors extracts dominant colors from an image (simplified)
func (p *ImageProcessor) extractDominantColors(img image.Image) []string {
	// Resize image for faster processing
	resized := imaging.Resize(img, 100, 0, imaging.Lanczos)

	colorMap := make(map[string]int)
	bounds := resized.Bounds()

	// Sample colors
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 5 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 5 {
			r, g, b, _ := resized.At(x, y).RGBA()
			// Convert to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// Quantize colors to reduce palette
			r8 = (r8 / 32) * 32
			g8 = (g8 / 32) * 32
			b8 = (b8 / 32) * 32

			color := fmt.Sprintf("#%02x%02x%02x", r8, g8, b8)
			colorMap[color]++
		}
	}

	// Find top 5 colors
	type colorCount struct {
		color string
		count int
	}

	var colors []colorCount
	for color, count := range colorMap {
		colors = append(colors, colorCount{color, count})
	}

	// Simple sort by count (bubble sort for simplicity)
	for i := 0; i < len(colors); i++ {
		for j := i + 1; j < len(colors); j++ {
			if colors[j].count > colors[i].count {
				colors[i], colors[j] = colors[j], colors[i]
			}
		}
	}

	// Return top 5
	result := make([]string, 0, 5)
	for i := 0; i < len(colors) && i < 5; i++ {
		result = append(result, colors[i].color)
	}

	return result
}

// GenerateThumbnail creates a thumbnail from an image
func (p *ImageProcessor) GenerateThumbnail(sourcePath, destPath string, width, height int) error {
	src, err := imaging.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Create thumbnail maintaining aspect ratio
	thumb := imaging.Fit(src, width, height, imaging.Lanczos)

	// Save thumbnail
	if err := imaging.Save(thumb, destPath); err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return nil
}
