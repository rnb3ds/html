package internal

import (
	"strings"
)

const (
	// Video and audio MIME types
	mimeMP4       = "video/mp4"
	mimeWebM      = "video/webm"
	mimeOGGVideo  = "video/ogg"
	mimeQuicktime = "video/quicktime"
	mimeAVI       = "video/x-msvideo"
	mimeWMV       = "video/x-ms-wmv"
	mimeFLV       = "video/x-flv"
	mimeMKV       = "video/x-matroska"
	mime3GP       = "video/3gpp"

	mimeMPEG  = "audio/mpeg"
	mimeWAV   = "audio/wav"
	mimeOGG   = "audio/ogg"
	mimeM4A   = "audio/mp4"
	mimeAAC   = "audio/aac"
	mimeFLAC  = "audio/flac"
	mimeWMA   = "audio/x-ms-wma"
	mimeOpus  = "audio/opus"
	mimeEmbed = "embed"
)

var (
	// Video extensions for video-specific detection
	videoExtensions = map[string]string{
		".mp4": mimeMP4, ".m4v": mimeMP4, ".webm": mimeWebM,
		".ogg": mimeOGGVideo, ".mov": mimeQuicktime, ".avi": mimeAVI,
		".wmv": mimeWMV, ".flv": mimeFLV, ".mkv": mimeMKV,
		".3gp": mime3GP,
	}

	// Audio extensions for audio-specific detection
	audioExtensions = map[string]string{
		".mp3": mimeMPEG, ".wav": mimeWAV, ".ogg": mimeOGG,
		".oga": mimeOGG, ".m4a": mimeM4A, ".aac": mimeAAC,
		".flac": mimeFLAC, ".wma": mimeWMA, ".opus": mimeOpus,
	}

	embedPatterns = []string{
		"youtube.com/embed/",
		"youtube-nocookie.com/embed/",
		"player.vimeo.com/video/",
		"dailymotion.com/embed/",
		"player.youku.com/",
		"v.qq.com/",
		"bilibili.com/",
	}
)

// IsVideoURL checks if a URL is a video based on extension or embed pattern
func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)
	return detectVideoType(lowerURL) != "" || hasEmbedPattern(lowerURL)
}

// DetectVideoType detects the video MIME type from a URL
func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	if mimeType := detectVideoType(lowerURL); mimeType != "" {
		return mimeType
	}
	if hasEmbedPattern(lowerURL) {
		return mimeEmbed
	}
	return ""
}

// DetectAudioType detects the audio MIME type from a URL
func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	return detectAudioType(lowerURL)
}

// detectVideoType performs lookup for video extensions
func detectVideoType(url string) string {
	for ext, mimeType := range videoExtensions {
		if strings.HasSuffix(url, ext) {
			return mimeType
		}
	}
	return ""
}

// detectAudioType performs lookup for audio extensions
func detectAudioType(url string) string {
	for ext, mimeType := range audioExtensions {
		if strings.HasSuffix(url, ext) {
			return mimeType
		}
	}
	return ""
}

// hasEmbedPattern checks if URL contains known embed patterns
func hasEmbedPattern(url string) bool {
	for _, pattern := range embedPatterns {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}
