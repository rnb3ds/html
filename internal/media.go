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
	videoExtensions = map[string]string{
		".mp4": mimeMP4, ".m4v": mimeMP4, ".webm": mimeWebM,
		".ogg": mimeOGGVideo, ".mov": mimeQuicktime, ".avi": mimeAVI,
		".wmv": mimeWMV, ".flv": mimeFLV, ".mkv": mimeMKV,
		".3gp": mime3GP,
	}

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

func IsVideoURL(url string) bool {
	lowerURL := strings.ToLower(url)
	return hasVideoExtension(lowerURL) || hasEmbedPattern(lowerURL)
}

func DetectVideoType(url string) string {
	lowerURL := strings.ToLower(url)
	if ext, ok := findVideoExtension(lowerURL); ok {
		return ext
	}
	if hasEmbedPattern(lowerURL) {
		return mimeEmbed
	}
	return ""
}

func DetectAudioType(url string) string {
	lowerURL := strings.ToLower(url)
	if ext, ok := findAudioExtension(lowerURL); ok {
		return ext
	}
	return ""
}

func hasVideoExtension(url string) bool {
	for ext := range videoExtensions {
		if strings.HasSuffix(url, ext) {
			return true
		}
	}
	return false
}

func hasEmbedPattern(url string) bool {
	for _, pattern := range embedPatterns {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

func findVideoExtension(url string) (string, bool) {
	for ext, mimeType := range videoExtensions {
		if strings.HasSuffix(url, ext) {
			return mimeType, true
		}
	}
	return "", false
}

func findAudioExtension(url string) (string, bool) {
	for ext, mimeType := range audioExtensions {
		if strings.HasSuffix(url, ext) {
			return mimeType, true
		}
	}
	return "", false
}
