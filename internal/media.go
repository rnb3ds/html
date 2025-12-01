package internal

import (
	"strings"
)

var embedPatterns = []string{
	"youtube.com/embed/",
	"youtube-nocookie.com/embed/",
	"player.vimeo.com/video/",
	"dailymotion.com/embed/",
	"player.youku.com/",
	"v.qq.com/",
	"bilibili.com/",
}

var videoExtensions = []string{".mp4", ".webm", ".ogg", ".mov", ".avi", ".wmv", ".flv", ".mkv", ".m4v", ".3gp"}

func IsVideoEmbedURL(url string) bool {
	lowerURL := strings.ToLower(url)
	for _, pattern := range embedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}
	return false
}

func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)
	for _, ext := range videoExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return IsVideoEmbedURL(url)
}

var videoTypeMap = map[string]string{
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".webm": "video/webm",
	".ogg":  "video/ogg",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".mkv":  "video/x-matroska",
	".3gp":  "video/3gpp",
}

func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	for ext, mimeType := range videoTypeMap {
		if strings.HasSuffix(lowerURL, ext) {
			return mimeType
		}
	}
	if IsVideoEmbedURL(url) {
		return "embed"
	}
	return ""
}

var audioTypeMap = map[string]string{
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".flac": "audio/flac",
	".wma":  "audio/x-ms-wma",
	".opus": "audio/opus",
}

func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	for ext, mimeType := range audioTypeMap {
		if strings.HasSuffix(lowerURL, ext) {
			return mimeType
		}
	}
	return ""
}
