package adapter

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
)

// Color represents an RGB color
type Color struct {
	R, G, B uint8
}

// ANSI color constants for 256-color mode
const (
	// Light pastel colors for better readability
	ColorLightBlue       = "\x1b[38;5;117m" // Light sky blue
	ColorLightGreen      = "\x1b[38;5;156m" // Light mint green
	ColorLightYellow     = "\x1b[38;5;229m" // Light yellow
	ColorLightOrange     = "\x1b[38;5;216m" // Light peach
	ColorLightPink       = "\x1b[38;5;218m" // Light pink
	ColorLightPurple     = "\x1b[38;5;183m" // Light lavender
	ColorLightCyan       = "\x1b[38;5;159m" // Light aqua
	ColorLightMagenta    = "\x1b[38;5;219m" // Light magenta
	ColorLightCoral      = "\x1b[38;5;217m" // Light coral
	ColorLightTeal       = "\x1b[38;5;123m" // Light teal
	ColorLightLime       = "\x1b[38;5;192m" // Light lime
	ColorLightSalmon     = "\x1b[38;5;223m" // Light salmon
	ColorLightPeriwinkle = "\x1b[38;5;147m" // Light periwinkle
	ColorLightMint       = "\x1b[38;5;194m" // Light mint
	ColorLightRose       = "\x1b[38;5;224m" // Light rose
	ColorLightSky        = "\x1b[38;5;153m" // Light sky
	ColorLightLavender   = "\x1b[38;5;189m" // Light lavender blue
	ColorLightPeach      = "\x1b[38;5;223m" // Light peach
	ColorLightTurquoise  = "\x1b[38;5;116m" // Light turquoise
	ColorLightGold       = "\x1b[38;5;222m" // Light gold

	// Softer versions of standard colors
	ColorSoftRed     = "\x1b[38;5;210m" // Soft red
	ColorSoftGreen   = "\x1b[38;5;114m" // Soft green
	ColorSoftBlue    = "\x1b[38;5;111m" // Soft blue
	ColorSoftYellow  = "\x1b[38;5;222m" // Soft yellow
	ColorSoftMagenta = "\x1b[38;5;176m" // Soft magenta
	ColorSoftCyan    = "\x1b[38;5;116m" // Soft cyan
	ColorSoftOrange  = "\x1b[38;5;215m" // Soft orange
	ColorSoftPurple  = "\x1b[38;5;141m" // Soft purple

	// Very light grays for subtle text
	ColorVeryLightGray = "\x1b[38;5;252m"
	ColorLightGray     = "\x1b[38;5;248m"
	ColorMediumGray    = "\x1b[38;5;244m"
	ColorDarkGray      = "\x1b[38;5;240m"

	// Background colors (light)
	BgLightBlue   = "\x1b[48;5;195m"
	BgLightGreen  = "\x1b[48;5;194m"
	BgLightYellow = "\x1b[48;5;230m"
	BgLightPink   = "\x1b[48;5;225m"
	BgLightGray   = "\x1b[48;5;254m"
)

// ColorPalette represents a collection of colors for a theme
type ColorPalette struct {
	Primary   []string
	Secondary []string
	Accent    []string
	Neutral   []string
	Status    map[string]string
	Semantic  map[string]string
}

// DefaultLightPalette returns a light, readable color palette
func DefaultLightPalette() ColorPalette {
	return ColorPalette{
		Primary: []string{
			ColorLightBlue,
			ColorLightGreen,
			ColorLightPurple,
			ColorLightCyan,
			ColorLightTeal,
		},
		Secondary: []string{
			ColorLightYellow,
			ColorLightOrange,
			ColorLightPink,
			ColorLightCoral,
			ColorLightRose,
		},
		Accent: []string{
			ColorLightPeriwinkle,
			ColorLightMint,
			ColorLightSky,
			ColorLightLavender,
			ColorLightTurquoise,
		},
		Neutral: []string{
			ColorVeryLightGray,
			ColorLightGray,
			ColorMediumGray,
		},
		Status: map[string]string{
			"success": ColorSoftGreen,
			"info":    ColorSoftBlue,
			"warning": ColorSoftYellow,
			"error":   ColorSoftRed,
			"debug":   ColorMediumGray,
		},
		Semantic: map[string]string{
			"key":        ColorLightBlue,
			"value":      ColorLightGreen,
			"string":     ColorLightGreen,
			"number":     ColorLightOrange,
			"boolean":    ColorLightPink,
			"null":       ColorMediumGray,
			"type":       ColorLightPurple,
			"error":      ColorSoftRed,
			"bracket":    ColorVeryLightGray,
			"comma":      ColorVeryLightGray,
			"colon":      ColorVeryLightGray,
			"quote":      ColorLightGray,
			"fieldName":  ColorLightCyan,
			"structName": ColorLightPeriwinkle,
		},
	}
}

// Colorizer provides advanced colorization capabilities
type Colorizer struct {
	enabled bool
	palette ColorPalette
	cache   map[string]string
}

// NewColorizer creates a new colorizer
func NewColorizer(enabled bool) *Colorizer {
	return &Colorizer{
		enabled: enabled,
		palette: DefaultLightPalette(),
		cache:   make(map[string]string),
	}
}

// Wrap wraps text with the given color
func (c *Colorizer) Wrap(text, color string) string {
	if !c.enabled {
		return text
	}
	return color + text + ansiReset
}

// WrapKey wraps a key with appropriate coloring
func (c *Colorizer) WrapKey(key string) string {
	return c.Wrap(key, c.palette.Semantic["key"])
}

// WrapValue wraps a value with appropriate coloring based on type
func (c *Colorizer) WrapValue(value string, valueType string) string {
	if color, ok := c.palette.Semantic[valueType]; ok {
		return c.Wrap(value, color)
	}
	return value
}

// WrapStatus wraps status text with appropriate coloring
func (c *Colorizer) WrapStatus(status string, statusType string) string {
	if color, ok := c.palette.Status[statusType]; ok {
		return c.Wrap(status, color)
	}
	return status
}

// HashColor returns a consistent color for a given string
func (c *Colorizer) HashColor(s string) string {
	if !c.enabled {
		return ""
	}

	if color, ok := c.cache[s]; ok {
		return color
	}

	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()

	colors := c.palette.Primary
	color := colors[hash%uint32(len(colors))]
	c.cache[s] = color

	return color
}

// Gradient creates a gradient effect across text
func (c *Colorizer) Gradient(text string, startColor, endColor Color) string {
	if !c.enabled || len(text) == 0 {
		return text
	}

	var result strings.Builder
	length := len(text)

	for i, char := range text {
		progress := float64(i) / float64(length-1)
		if length == 1 {
			progress = 0
		}

		r := uint8(float64(startColor.R) + progress*float64(int(endColor.R)-int(startColor.R)))
		g := uint8(float64(startColor.G) + progress*float64(int(endColor.G)-int(startColor.G)))
		b := uint8(float64(startColor.B) + progress*float64(int(endColor.B)-int(startColor.B)))

		result.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm%c", r, g, b, char))
	}

	result.WriteString(ansiReset)
	return result.String()
}

// RainbowText applies a rainbow effect to text
func (c *Colorizer) RainbowText(text string) string {
	if !c.enabled || len(text) == 0 {
		return text
	}

	rainbowColors := []string{
		ColorSoftRed,
		ColorSoftOrange,
		ColorSoftYellow,
		ColorSoftGreen,
		ColorSoftCyan,
		ColorSoftBlue,
		ColorSoftPurple,
	}

	var result strings.Builder
	colorIndex := 0

	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			result.WriteRune(char)
			continue
		}

		result.WriteString(rainbowColors[colorIndex%len(rainbowColors)])
		result.WriteRune(char)
		result.WriteString(ansiReset)
		colorIndex++
	}

	return result.String()
}

// PulseEffect creates a pulsing brightness effect
func (c *Colorizer) PulseEffect(text string, baseColor Color, intensity float64) string {
	if !c.enabled {
		return text
	}

	pulse := math.Sin(intensity * math.Pi)
	brightness := 0.7 + (pulse * 0.3)

	r := uint8(math.Min(255, float64(baseColor.R)*brightness))
	g := uint8(math.Min(255, float64(baseColor.G)*brightness))
	b := uint8(math.Min(255, float64(baseColor.B)*brightness))

	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s%s", r, g, b, text, ansiReset)
}

// HighlightBrackets highlights matching brackets in different colors
func (c *Colorizer) HighlightBrackets(text string) string {
	if !c.enabled {
		return text
	}

	bracketColors := []string{
		ColorLightBlue,
		ColorLightGreen,
		ColorLightPurple,
		ColorLightCyan,
		ColorLightOrange,
	}

	var result strings.Builder
	depth := 0
	inString := false

	for _, char := range text {
		if char == '"' && !inString {
			inString = true
			result.WriteRune(char)
			continue
		} else if char == '"' && inString {
			inString = false
			result.WriteRune(char)
			continue
		}

		if inString {
			result.WriteRune(char)
			continue
		}

		switch char {
		case '{', '[', '(':
			color := bracketColors[depth%len(bracketColors)]
			result.WriteString(color)
			result.WriteRune(char)
			result.WriteString(ansiReset)
			depth++
		case '}', ']', ')':
			depth--
			if depth < 0 {
				depth = 0
			}
			color := bracketColors[depth%len(bracketColors)]
			result.WriteString(color)
			result.WriteRune(char)
			result.WriteString(ansiReset)
		default:
			result.WriteRune(char)
		}
	}

	return result.String()
}

// ColorizeJSON colorizes JSON output with syntax highlighting
func (c *Colorizer) ColorizeJSON(jsonStr string) string {
	if !c.enabled {
		return jsonStr
	}

	var result strings.Builder
	inString := false
	inNumber := false
	afterColon := false

	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]

		// Handle strings
		if char == '"' {
			if !inString {
				inString = true
				if afterColon {
					result.WriteString(c.palette.Semantic["string"])
				} else {
					result.WriteString(c.palette.Semantic["key"])
				}
				result.WriteByte(char)
				continue
			} else {
				result.WriteByte(char)
				result.WriteString(ansiReset)
				inString = false
				afterColon = false
				continue
			}
		}

		if inString {
			result.WriteByte(char)
			continue
		}

		// Handle numbers
		if (char >= '0' && char <= '9') || char == '-' || char == '.' || char == 'e' || char == 'E' {
			if !inNumber {
				result.WriteString(c.palette.Semantic["number"])
				inNumber = true
			}
			result.WriteByte(char)
			continue
		} else if inNumber {
			result.WriteString(ansiReset)
			inNumber = false
		}

		// Handle special keywords
		if i+4 <= len(jsonStr) && jsonStr[i:i+4] == "null" {
			result.WriteString(c.Wrap("null", c.palette.Semantic["null"]))
			i += 3
			continue
		}
		if i+4 <= len(jsonStr) && jsonStr[i:i+4] == "true" {
			result.WriteString(c.Wrap("true", c.palette.Semantic["boolean"]))
			i += 3
			continue
		}
		if i+5 <= len(jsonStr) && jsonStr[i:i+5] == "false" {
			result.WriteString(c.Wrap("false", c.palette.Semantic["boolean"]))
			i += 4
			continue
		}

		// Handle punctuation
		switch char {
		case '{', '}', '[', ']':
			result.WriteString(c.Wrap(string(char), c.palette.Semantic["bracket"]))
		case ':':
			result.WriteString(c.Wrap(string(char), c.palette.Semantic["colon"]))
			afterColon = true
		case ',':
			result.WriteString(c.Wrap(string(char), c.palette.Semantic["comma"]))
			afterColon = false
		case ' ', '\n', '\t', '\r':
			result.WriteByte(char)
		default:
			result.WriteByte(char)
		}
	}

	if inNumber {
		result.WriteString(ansiReset)
	}

	return result.String()
}

// BoxDrawing provides box-drawing utilities with colors
type BoxDrawing struct {
	colorizer *Colorizer
	style     BoxStyle
}

// BoxStyle defines the style of box drawing
type BoxStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	TeeLeft     string
	TeeRight    string
	TeeTop      string
	TeeBottom   string
	Cross       string
	Color       string
}

// LightBoxStyle returns a light box style
func LightBoxStyle() BoxStyle {
	return BoxStyle{
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
		Horizontal:  "─",
		Vertical:    "│",
		TeeLeft:     "├",
		TeeRight:    "┤",
		TeeTop:      "┬",
		TeeBottom:   "┴",
		Cross:       "┼",
		Color:       ColorVeryLightGray,
	}
}

// HeavyBoxStyle returns a heavy box style
func HeavyBoxStyle() BoxStyle {
	return BoxStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		TeeLeft:     "╠",
		TeeRight:    "╣",
		TeeTop:      "╦",
		TeeBottom:   "╩",
		Cross:       "╬",
		Color:       ColorLightBlue,
	}
}

// DoubleBoxStyle returns a double-line box style
func DoubleBoxStyle() BoxStyle {
	return BoxStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		TeeLeft:     "╠",
		TeeRight:    "╣",
		TeeTop:      "╦",
		TeeBottom:   "╩",
		Cross:       "╬",
		Color:       ColorLightCyan,
	}
}

// RoundedBoxStyle returns a rounded box style
func RoundedBoxStyle() BoxStyle {
	return BoxStyle{
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
		Horizontal:  "─",
		Vertical:    "│",
		TeeLeft:     "├",
		TeeRight:    "┤",
		TeeTop:      "┬",
		TeeBottom:   "┴",
		Cross:       "┼",
		Color:       ColorLightPurple,
	}
}

// NewBoxDrawing creates a new box drawing helper
func NewBoxDrawing(colorizer *Colorizer, style BoxStyle) *BoxDrawing {
	return &BoxDrawing{
		colorizer: colorizer,
		style:     style,
	}
}

// DrawBox draws a box around content
func (bd *BoxDrawing) DrawBox(content string, width int, title string) string {
	lines := strings.Split(content, "\n")
	var result strings.Builder

	// Top border
	result.WriteString(bd.colorize(bd.style.TopLeft))
	if title != "" {
		titlePadding := width - len(title) - 2
		if titlePadding > 0 {
			result.WriteString(bd.colorize(strings.Repeat(bd.style.Horizontal, 1)))
			result.WriteString(" ")
			result.WriteString(bd.colorizer.Wrap(title, ColorLightPeriwinkle))
			result.WriteString(" ")
			result.WriteString(bd.colorize(strings.Repeat(bd.style.Horizontal, titlePadding-2)))
		} else {
			result.WriteString(bd.colorize(strings.Repeat(bd.style.Horizontal, width)))
		}
	} else {
		result.WriteString(bd.colorize(strings.Repeat(bd.style.Horizontal, width)))
	}
	result.WriteString(bd.colorize(bd.style.TopRight))
	result.WriteString("\n")

	// Content
	for _, line := range lines {
		result.WriteString(bd.colorize(bd.style.Vertical))
		result.WriteString(" ")
		result.WriteString(line)

		// Pad to width
		visibleLen := MeasureVisibleLength(line)
		if visibleLen < width-2 {
			result.WriteString(strings.Repeat(" ", width-2-visibleLen))
		}

		result.WriteString(" ")
		result.WriteString(bd.colorize(bd.style.Vertical))
		result.WriteString("\n")
	}

	// Bottom border
	result.WriteString(bd.colorize(bd.style.BottomLeft))
	result.WriteString(bd.colorize(strings.Repeat(bd.style.Horizontal, width)))
	result.WriteString(bd.colorize(bd.style.BottomRight))
	result.WriteString("\n")

	return result.String()
}

// DrawHorizontalLine draws a horizontal line
func (bd *BoxDrawing) DrawHorizontalLine(width int) string {
	return bd.colorize(bd.style.TeeLeft) +
		bd.colorize(strings.Repeat(bd.style.Horizontal, width)) +
		bd.colorize(bd.style.TeeRight) + "\n"
}

// colorize applies the box style color
func (bd *BoxDrawing) colorize(s string) string {
	return bd.colorizer.Wrap(s, bd.style.Color)
}

// ProgressBar represents a colored progress bar
type ProgressBar struct {
	width       int
	fillChar    string
	emptyChar   string
	fillColor   string
	emptyColor  string
	borderColor string
	colorizer   *Colorizer
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int, colorizer *Colorizer) *ProgressBar {
	return &ProgressBar{
		width:       width,
		fillChar:    "█",
		emptyChar:   "░",
		fillColor:   ColorLightGreen,
		emptyColor:  ColorLightGray,
		borderColor: ColorVeryLightGray,
		colorizer:   colorizer,
	}
}

// Render renders the progress bar
func (pb *ProgressBar) Render(percentage float64) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	filled := int(float64(pb.width) * percentage / 100)
	empty := pb.width - filled

	var result strings.Builder
	result.WriteString(pb.colorizer.Wrap("[", pb.borderColor))
	result.WriteString(pb.colorizer.Wrap(strings.Repeat(pb.fillChar, filled), pb.fillColor))
	result.WriteString(pb.colorizer.Wrap(strings.Repeat(pb.emptyChar, empty), pb.emptyColor))
	result.WriteString(pb.colorizer.Wrap("]", pb.borderColor))
	result.WriteString(fmt.Sprintf(" %.1f%%", percentage))

	return result.String()
}

// StatusIndicator provides status indicators with colors
type StatusIndicator struct {
	colorizer *Colorizer
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator(colorizer *Colorizer) *StatusIndicator {
	return &StatusIndicator{colorizer: colorizer}
}

// Success returns a success indicator
func (si *StatusIndicator) Success() string {
	return si.colorizer.Wrap("✓", ColorSoftGreen)
}

// Error returns an error indicator
func (si *StatusIndicator) Error() string {
	return si.colorizer.Wrap("✗", ColorSoftRed)
}

// Warning returns a warning indicator
func (si *StatusIndicator) Warning() string {
	return si.colorizer.Wrap("⚠", ColorSoftYellow)
}

// Info returns an info indicator
func (si *StatusIndicator) Info() string {
	return si.colorizer.Wrap("ℹ", ColorSoftBlue)
}

// Debug returns a debug indicator
func (si *StatusIndicator) Debug() string {
	return si.colorizer.Wrap("🐛", ColorMediumGray)
}

// Spinner returns a spinner character
func (si *StatusIndicator) Spinner(frame int) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return si.colorizer.Wrap(spinners[frame%len(spinners)], ColorLightCyan)
}
