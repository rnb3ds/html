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

// Unified media type registry
var mediaTypes = []MediaType{
	// Video types
	{".mp4", "video/mp4", "video"},
	{".m4v", "video/mp4", "video"},
	{".webm", "video/webm", "video"},
	{".ogg", "video/ogg", "video"},
	{".mov", "video/quicktime", "video"},
	{".avi", "video/x-msvideo", "video"},
	{".wmv", "video/x-ms-wmv", "video"},
	{".flv", "video/x-flv", "video"},
	{".mkv", "video/x-matroska", "video"},
	{".3gp", "video/3gpp", "video"},

	// Audio types
	{".mp3", "audio/mpeg", "audio"},
	{".wav", "audio/wav", "audio"},
	{".ogg", "audio/ogg", "audio"},
	{".oga", "audio/ogg", "audio"},
	{".m4a", "audio/mp4", "audio"},
	{".aac", "audio/aac", "audio"},
	{".flac", "audio/flac", "audio"},
	{".wma", "audio/x-ms-wma", "audio"},
	{".opus", "audio/opus", "audio"},
	{".oga", "audio/ogg", "audio"},
	{".m4a", "audio/mp4", "audio"},
	{".aac", "audio/aac", "audio"},
	{".flac", "audio/flac", "audio"},
	{".wma", "audio/x-ms-wma", "audio"},
	{".opus", "audio/opus", "audio"},
}

// Video embed patterns for platform detection
var embedPatterns = []string{
	"youtube.com/embed/",
	"youtube-nocookie.com/embed/",
	"player.vimeo.com/video/",
	"dailymotion.com/embed/",
	"player.youku.com/",
	"v.qq.com/",
	"bilibili.com/",
}

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

// IsVideoURL checks if URL is a video file or embed.
func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)
	for _, media := range mediaTypes {
		if media.Category == "video" && strings.HasSuffix(lowerURL, media.Extension) {
			return true
		}
	}
	return IsVideoEmbedURL(url)
}

// DetectVideoType returns MIME type for video URLs.
func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	for _, media := range mediaTypes {
		if media.Category == "video" && strings.HasSuffix(lowerURL, media.Extension) {
			return media.MimeType
		}
	}
	if IsVideoEmbedURL(url) {
		return "embed"
	}
	return ""
}

// DetectAudioType returns MIME type for audio URLs.
func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	for _, media := range mediaTypes {
		if media.Category == "audio" && strings.HasSuffix(lowerURL, media.Extension) {
			return media.MimeType
		}
	}
	return ""
}
