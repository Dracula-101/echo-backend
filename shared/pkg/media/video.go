package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type VideoProcessor struct{}

func NewVideoProcessor() *VideoProcessor {
	return &VideoProcessor{}
}

type FFprobeOutput struct {
	Streams []FFprobeStream `json:"streams"`
	Format  FFprobeFormat   `json:"format"`
}

type FFprobeStream struct {
	Index              int    `json:"index"`
	CodecName          string `json:"codec_name"`
	CodecLongName      string `json:"codec_long_name"`
	CodecType          string `json:"codec_type"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	DisplayAspectRatio string `json:"display_aspect_ratio"`
	AvgFrameRate       string `json:"avg_frame_rate"`
	BitRate            string `json:"bit_rate"`
	Channels           int    `json:"channels"`
	SampleRate         string `json:"sample_rate"`
	Duration           string `json:"duration"`
	Tags               struct {
		Language string `json:"language"`
		Title    string `json:"title"`
	} `json:"tags"`
}

type FFprobeFormat struct {
	Filename   string            `json:"filename"`
	FormatName string            `json:"format_name"`
	Duration   string            `json:"duration"`
	Size       string            `json:"size"`
	BitRate    string            `json:"bit_rate"`
	Tags       map[string]string `json:"tags"`
}

func (p *VideoProcessor) Process(ctx context.Context, filePath string, info *MediaInfo) error {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe not found in PATH (install ffmpeg): %w", err)
	}

	// Run ffprobe to get video metadata
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var probe FFprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		info.DurationSeconds = &duration
	}

	if bitrate, err := strconv.Atoi(probe.Format.BitRate); err == nil {
		info.Bitrate = &bitrate
	}

	var videoStream *FFprobeStream
	var audioStream *FFprobeStream
	subtitleCount := 0

	for i := range probe.Streams {
		stream := &probe.Streams[i]
		switch stream.CodecType {
		case "video":
			if videoStream == nil {
				videoStream = stream
			}
		case "audio":
			if audioStream == nil {
				audioStream = stream
			}
		case "subtitle":
			subtitleCount++
		}
	}

	// Extract video stream information
	if videoStream != nil {
		info.Width = &videoStream.Width
		info.Height = &videoStream.Height

		if videoStream.CodecName != "" {
			info.VideoCodec = &videoStream.CodecName
			info.Codec = &videoStream.CodecName
		}

		if videoStream.AvgFrameRate != "" {
			frameRate := parseFrameRate(videoStream.AvgFrameRate)
			if frameRate > 0 {
				info.FrameRate = &frameRate
			}
		}

		if videoStream.Width > 0 && videoStream.Height > 0 {
			resolution := fmt.Sprintf("%dx%d", videoStream.Width, videoStream.Height)
			info.Resolution = &resolution

			aspectRatio := CalculateAspectRatio(videoStream.Width, videoStream.Height)
			info.AspectRatio = &aspectRatio
		}

		if videoStream.DisplayAspectRatio != "" {
			info.AspectRatio = &videoStream.DisplayAspectRatio
		}
	}

	if audioStream != nil {
		if audioStream.CodecName != "" {
			info.AudioCodec = &audioStream.CodecName
		}

		if audioStream.Channels > 0 {
			info.AudioChannels = &audioStream.Channels
		}

		if audioStream.SampleRate != "" {
			if sampleRate, err := strconv.Atoi(audioStream.SampleRate); err == nil {
				info.SampleRate = &sampleRate
			}
		}
	}

	if subtitleCount > 0 {
		info.SubtitleTracks = &subtitleCount
	}

	return nil
}

func parseFrameRate(frameRateStr string) float64 {
	parts := strings.Split(frameRateStr, "/")
	if len(parts) != 2 {
		if fps, err := strconv.ParseFloat(frameRateStr, 64); err == nil {
			return fps
		}
		return 0
	}

	num, err1 := strconv.ParseFloat(parts[0], 64)
	denom, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || denom == 0 {
		return 0
	}

	return num / denom
}

func (p *VideoProcessor) ExtractThumbnail(ctx context.Context, videoPath, outputPath string, timeSeconds float64) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", videoPath,
		"-ss", fmt.Sprintf("%.2f", timeSeconds),
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-y", // Overwrite output file
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract thumbnail: %w", err)
	}

	return nil
}

func (p *VideoProcessor) ExtractMultipleThumbnails(ctx context.Context, videoPath, outputDir string, count int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video duration: %w", err)
	}

	var probe FFprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	duration, err := strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	var thumbnails []string
	interval := duration / float64(count+1)

	for i := 1; i <= count; i++ {
		timestamp := interval * float64(i)
		outputPath := fmt.Sprintf("%s/thumb_%d.jpg", outputDir, i)

		if err := p.ExtractThumbnail(ctx, videoPath, outputPath, timestamp); err != nil {
			return thumbnails, err
		}

		thumbnails = append(thumbnails, outputPath)
	}

	return thumbnails, nil
}
