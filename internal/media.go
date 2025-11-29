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

func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	if strings.HasSuffix(lowerURL, ".mp4") || strings.HasSuffix(lowerURL, ".m4v") {
		return "video/mp4"
	} else if strings.HasSuffix(lowerURL, ".webm") {
		return "video/webm"
	} else if strings.HasSuffix(lowerURL, ".ogg") {
		return "video/ogg"
	} else if strings.HasSuffix(lowerURL, ".mov") {
		return "video/quicktime"
	} else if strings.HasSuffix(lowerURL, ".avi") {
		return "video/x-msvideo"
	} else if strings.HasSuffix(lowerURL, ".wmv") {
		return "video/x-ms-wmv"
	} else if strings.HasSuffix(lowerURL, ".flv") {
		return "video/x-flv"
	} else if strings.HasSuffix(lowerURL, ".mkv") {
		return "video/x-matroska"
	} else if strings.HasSuffix(lowerURL, ".3gp") {
		return "video/3gpp"
	} else if IsVideoEmbedURL(url) {
		return "embed"
	}
	return ""
}

func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	if strings.HasSuffix(lowerURL, ".mp3") {
		return "audio/mpeg"
	} else if strings.HasSuffix(lowerURL, ".wav") {
		return "audio/wav"
	} else if strings.HasSuffix(lowerURL, ".ogg") || strings.HasSuffix(lowerURL, ".oga") {
		return "audio/ogg"
	} else if strings.HasSuffix(lowerURL, ".m4a") {
		return "audio/mp4"
	} else if strings.HasSuffix(lowerURL, ".aac") {
		return "audio/aac"
	} else if strings.HasSuffix(lowerURL, ".flac") {
		return "audio/flac"
	} else if strings.HasSuffix(lowerURL, ".wma") {
		return "audio/x-ms-wma"
	} else if strings.HasSuffix(lowerURL, ".opus") {
		return "audio/opus"
	}
	return ""
}
