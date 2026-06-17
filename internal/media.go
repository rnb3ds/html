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

	// mediaPatterns groups every media signature — file extensions (".mp4", ".mp3",
	// ...) and embed-host patterns ("youtube.com/embed/", ...) — by its first byte
	// (lowercased). HasMediaReference uses it to find any signature in a single
	// allocation-free pass, checking only the few signatures that start with the
	// byte at the current position instead of scanning once per pattern.
	mediaPatterns [256][]string
)

func init() {
	addSignature := func(sig string) {
		if sig == "" {
			return
		}
		first := sig[0]
		if first >= 'A' && first <= 'Z' {
			first += 32
		}
		mediaPatterns[first] = append(mediaPatterns[first], sig)
	}
	for ext := range videoExtensions {
		addSignature(ext)
	}
	for ext := range audioExtensions {
		addSignature(ext)
	}
	for _, pattern := range embedPatterns {
		addSignature(pattern)
	}
}

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

// detectVideoType performs lookup for video extensions.
// Handles URLs with query parameters and fragments by stripping them first.
func detectVideoType(url string) string {
	// Remove query parameters and fragments
	if idx := strings.IndexByte(url, '?'); idx >= 0 {
		url = url[:idx]
	}
	if idx := strings.IndexByte(url, '#'); idx >= 0 {
		url = url[:idx]
	}

	for ext, mimeType := range videoExtensions {
		if strings.HasSuffix(url, ext) {
			return mimeType
		}
	}
	return ""
}

// detectAudioType performs lookup for audio extensions.
// Handles URLs with query parameters and fragments by stripping them first.
func detectAudioType(url string) string {
	// Remove query parameters and fragments
	if idx := strings.IndexByte(url, '?'); idx >= 0 {
		url = url[:idx]
	}
	if idx := strings.IndexByte(url, '#'); idx >= 0 {
		url = url[:idx]
	}

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

// HasMediaReference reports whether content contains a byte sequence that could
// form a media URL: a recognized media file extension (".mp4", ".mp3", ...) or a
// known embed-host pattern ("youtube.com/embed/", ...). The scan is allocation-free
// and ASCII case-insensitive.
//
// It performs a single pass over the content, dispatching on the current byte to the
// small set of signatures that begin with that byte (see mediaPatterns). This finds
// both file extensions and embed-host patterns in one traversal.
//
// It is a *necessary condition* for the regex-based and raw-HTML media scans in the
// public package to produce any result: a video/audio regex match, or an
// iframe/embed/object source that resolves to a video, always contains one of these
// substrings. Callers therefore use a false result to skip those expensive scans with
// no change in output — a false result provably implies the scans would have been empty.
//
// A prefix (not suffix-delimited) match is used for extensions: the regex can match an
// extension even when immediately followed by other characters, so any occurrence must
// be treated as a potential match to avoid a false negative.
func HasMediaReference(content string) bool {
	n := len(content)
	for i := 0; i < n; i++ {
		c := content[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		bucket := mediaPatterns[c]
		if len(bucket) == 0 {
			continue
		}
		for _, sig := range bucket {
			if asciiFoldHasPrefix(content[i:], sig) {
				return true
			}
		}
	}
	return false
}

// asciiFoldHasPrefix reports whether s begins with prefix, ignoring ASCII case.
// prefix is assumed lowercase (the mediaPatterns entries are, by construction).
func asciiFoldHasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if c != prefix[i] {
			return false
		}
	}
	return true
}
