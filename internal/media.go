package internal

import (
	"strings"
)

var (
	videoExtensions = map[string]string{
		".mp4": "video/mp4", ".m4v": "video/mp4", ".webm": "video/webm",
		".ogg": "video/ogg", ".mov": "video/quicktime", ".avi": "video/x-msvideo",
		".wmv": "video/x-ms-wmv", ".flv": "video/x-flv", ".mkv": "video/x-matroska",
		".3gp": "video/3gpp",
	}

	audioExtensions = map[string]string{
		".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg",
		".oga": "audio/ogg", ".m4a": "audio/mp4", ".aac": "audio/aac",
		".flac": "audio/flac", ".wma": "audio/x-ms-wma", ".opus": "audio/opus",
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

func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)
	for _, pattern := range embedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}
	for ext := range videoExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return false
}

func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	for ext, mimeType := range videoExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return mimeType
		}
	}
	for _, pattern := range embedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return "embed"
		}
	}
	return ""
}

func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	for ext, mimeType := range audioExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return mimeType
		}
	}
	return ""
}
