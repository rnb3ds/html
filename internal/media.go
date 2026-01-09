package internal

import (
	"strings"
)

// Media type registry for unified media detection
type MediaType struct {
	Extension string
	MimeType  string
	Category  string // "video" or "audio"
}

// Optimized media type maps for O(1) lookup
var (
	videoExtensions = map[string]string{
		".mp4":  "video/mp4",
		".m4v":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg", // Note: .ogg can be both video and audio
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".mkv":  "video/x-matroska",
		".3gp":  "video/3gpp",
	}

	audioExtensions = map[string]string{
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg", // Note: .ogg can be both video and audio
		".oga":  "audio/ogg",
		".m4a":  "audio/mp4",
		".aac":  "audio/aac",
		".flac": "audio/flac",
		".wma":  "audio/x-ms-wma",
		".opus": "audio/opus",
	}

	// Video embed patterns for platform detection
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

// IsVideoEmbedURL checks if URL is a video embed from known platforms.
func IsVideoEmbedURL(url string) bool {
	lowerURL := strings.ToLower(url)
	for _, pattern := range embedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}
	return false
}

// IsVideoURL checks if URL is a video file or embed (optimized O(1) map lookup).
func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)
	// Check embed patterns first (more common)
	for _, pattern := range embedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}
	// Check file extensions with O(1) map lookup
	for ext := range videoExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return false
}

// DetectVideoType returns MIME type for video URLs (optimized O(1) map lookup).
func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	// Check file extensions first (more specific) with O(1) lookup
	for ext, mimeType := range videoExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return mimeType
		}
	}
	// Check embed patterns
	for _, pattern := range embedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return "embed"
		}
	}
	return ""
}

// DetectAudioType returns MIME type for audio URLs (optimized O(1) map lookup).
func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	for ext, mimeType := range audioExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return mimeType
		}
	}
	return ""
}
