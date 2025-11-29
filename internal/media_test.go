package internal

import (
	"testing"
)

func TestIsVideoEmbedURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{"https://youtube.com/embed/abc123", true},
		{"https://www.youtube.com/embed/abc123", true},
		{"https://youtube-nocookie.com/embed/abc123", true},
		{"https://player.vimeo.com/video/123456", true},
		{"https://dailymotion.com/embed/video123", true},
		{"https://player.youku.com/embed/abc", true},
		{"https://v.qq.com/iframe/player.html", true},
		{"https://bilibili.com/video/av123", true},
		{"https://example.com/video.mp4", false},
		{"https://example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := IsVideoEmbedURL(tt.url)
			if result != tt.want {
				t.Errorf("IsVideoEmbedURL(%q) = %v, want %v", tt.url, result, tt.want)
			}
		})
	}
}

func TestIsVideoURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/video.mp4", true},
		{"https://example.com/video.webm", true},
		{"https://example.com/video.ogg", true},
		{"https://example.com/video.mov", true},
		{"https://example.com/video.avi", true},
		{"https://example.com/video.wmv", true},
		{"https://example.com/video.flv", true},
		{"https://example.com/video.mkv", true},
		{"https://example.com/video.m4v", true},
		{"https://example.com/video.3gp", true},
		{"https://youtube.com/embed/abc", true},
		{"https://example.com/image.jpg", false},
		{"https://example.com/audio.mp3", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := IsVideoURL(tt.url)
			if result != tt.want {
				t.Errorf("IsVideoURL(%q) = %v, want %v", tt.url, result, tt.want)
			}
		})
	}
}

func TestDetectVideoType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/video.mp4", "video/mp4"},
		{"https://example.com/video.m4v", "video/mp4"},
		{"https://example.com/video.webm", "video/webm"},
		{"https://example.com/video.ogg", "video/ogg"},
		{"https://example.com/video.mov", "video/quicktime"},
		{"https://example.com/video.avi", "video/x-msvideo"},
		{"https://example.com/video.wmv", "video/x-ms-wmv"},
		{"https://example.com/video.flv", "video/x-flv"},
		{"https://example.com/video.mkv", "video/x-matroska"},
		{"https://example.com/video.3gp", "video/3gpp"},
		{"https://youtube.com/embed/abc", "embed"},
		{"https://example.com/unknown.xyz", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectVideoType(tt.url)
			if result != tt.want {
				t.Errorf("DetectVideoType(%q) = %q, want %q", tt.url, result, tt.want)
			}
		})
	}
}

func TestDetectAudioType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/audio.mp3", "audio/mpeg"},
		{"https://example.com/audio.wav", "audio/wav"},
		{"https://example.com/audio.ogg", "audio/ogg"},
		{"https://example.com/audio.oga", "audio/ogg"},
		{"https://example.com/audio.m4a", "audio/mp4"},
		{"https://example.com/audio.aac", "audio/aac"},
		{"https://example.com/audio.flac", "audio/flac"},
		{"https://example.com/audio.wma", "audio/x-ms-wma"},
		{"https://example.com/audio.opus", "audio/opus"},
		{"https://example.com/unknown.xyz", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectAudioType(tt.url)
			if result != tt.want {
				t.Errorf("DetectAudioType(%q) = %q, want %q", tt.url, result, tt.want)
			}
		})
	}
}

func TestCaseInsensitiveExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		fn   func(string) bool
	}{
		{"uppercase MP4", "https://example.com/VIDEO.MP4", IsVideoURL},
		{"mixed case WebM", "https://example.com/video.WeBm", IsVideoURL},
		{"uppercase MP3", "https://example.com/AUDIO.MP3", func(url string) bool {
			return DetectAudioType(url) != ""
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.fn(tt.url) {
				t.Errorf("Function should handle case-insensitive URL: %q", tt.url)
			}
		})
	}
}

func TestEmbedPatterns(t *testing.T) {
	t.Parallel()

	embedURLs := []string{
		"https://youtube.com/embed/abc123",
		"https://www.youtube.com/embed/abc123",
		"https://youtube-nocookie.com/embed/abc123",
		"https://player.vimeo.com/video/123456",
		"https://www.dailymotion.com/embed/video/x123",
		"https://player.youku.com/embed/abc",
		"https://v.qq.com/txp/iframe/player.html",
		"https://www.bilibili.com/blackboard/html5player.html",
	}

	for _, url := range embedURLs {
		t.Run(url, func(t *testing.T) {
			if !IsVideoEmbedURL(url) {
				t.Errorf("IsVideoEmbedURL(%q) should return true", url)
			}
			if !IsVideoURL(url) {
				t.Errorf("IsVideoURL(%q) should return true for embed", url)
			}
		})
	}
}

func BenchmarkIsVideoEmbedURL(b *testing.B) {
	url := "https://youtube.com/embed/abc123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsVideoEmbedURL(url)
	}
}

func BenchmarkIsVideoURL(b *testing.B) {
	url := "https://example.com/video.mp4"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsVideoURL(url)
	}
}

func BenchmarkDetectVideoType(b *testing.B) {
	url := "https://example.com/video.mp4"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectVideoType(url)
	}
}

func BenchmarkDetectAudioType(b *testing.B) {
	url := "https://example.com/audio.mp3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectAudioType(url)
	}
}
