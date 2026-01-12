package internal

import "testing"

// TestIsVideoURL tests video URL detection
func TestIsVideoURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "YouTube embed URL",
			url:  "https://www.youtube.com/embed/123456",
			want: true,
		},
		{
			name: "Vimeo embed URL",
			url:  "https://player.vimeo.com/video/123456",
			want: true,
		},
		{
			name: "Dailymotion embed URL",
			url:  "https://www.dailymotion.com/embed/video/123456",
			want: true,
		},
		{
			name: "MP4 file extension",
			url:  "https://example.com/video.MP4",
			want: true,
		},
		{
			name: "WebM file extension",
			url:  "https://example.com/video.webm",
			want: true,
		},
		{
			name: "OGG video extension",
			url:  "https://example.com/video.ogg",
			want: true,
		},
		{
			name: "MOV file extension",
			url:  "https://example.com/video.mov",
			want: true,
		},
		{
			name: "AVI file extension",
			url:  "https://example.com/video.avi",
			want: true,
		},
		{
			name: "WMV file extension",
			url:  "https://example.com/video.wmv",
			want: true,
		},
		{
			name: "FLV file extension",
			url:  "https://example.com/video.flv",
			want: true,
		},
		{
			name: "MKV file extension",
			url:  "https://example.com/video.mkv",
			want: true,
		},
		{
			name: "M4V file extension",
			url:  "https://example.com/video.m4v",
			want: true,
		},
		{
			name: "3GP file extension",
			url:  "https://example.com/video.3gp",
			want: true,
		},
		{
			name: "non-video URL",
			url:  "https://example.com/page.html",
			want: false,
		},
		{
			name: "image URL",
			url:  "https://example.com/image.jpg",
			want: false,
		},
		{
			name: "empty string",
			url:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVideoURL(tt.url); got != tt.want {
				t.Errorf("IsVideoURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetectVideoType tests video type detection
func TestDetectVideoType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "MP4 file",
			url:  "https://example.com/video.mp4",
			want: "video/mp4",
		},
		{
			name: "WebM file",
			url:  "https://example.com/video.webm",
			want: "video/webm",
		},
		{
			name: "OGG file",
			url:  "https://example.com/video.ogg",
			want: "video/ogg",
		},
		{
			name: "MOV file",
			url:  "https://example.com/video.mov",
			want: "video/quicktime",
		},
		{
			name: "AVI file",
			url:  "https://example.com/video.avi",
			want: "video/x-msvideo",
		},
		{
			name: "WMV file",
			url:  "https://example.com/video.wmv",
			want: "video/x-ms-wmv",
		},
		{
			name: "FLV file",
			url:  "https://example.com/video.flv",
			want: "video/x-flv",
		},
		{
			name: "MKV file",
			url:  "https://example.com/video.mkv",
			want: "video/x-matroska",
		},
		{
			name: "M4V file",
			url:  "https://example.com/video.m4v",
			want: "video/mp4",
		},
		{
			name: "3GP file",
			url:  "https://example.com/video.3gp",
			want: "video/3gpp",
		},
		{
			name: "YouTube embed",
			url:  "https://www.youtube.com/embed/123456",
			want: "embed",
		},
		{
			name: "Vimeo embed",
			url:  "https://player.vimeo.com/video/123456",
			want: "embed",
		},
		{
			name: "unknown video type",
			url:  "https://example.com/video.unknown",
			want: "",
		},
		{
			name: "non-video URL",
			url:  "https://example.com/page.html",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectVideoType(tt.url); got != tt.want {
				t.Errorf("DetectVideoType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetectAudioType tests audio type detection
func TestDetectAudioType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "MP3 file",
			url:  "https://example.com/audio.mp3",
			want: "audio/mpeg",
		},
		{
			name: "WAV file",
			url:  "https://example.com/audio.wav",
			want: "audio/wav",
		},
		{
			name: "OGG audio file",
			url:  "https://example.com/audio.ogg",
			want: "audio/ogg",
		},
		{
			name: "OGA file",
			url:  "https://example.com/audio.oga",
			want: "audio/ogg",
		},
		{
			name: "M4A file",
			url:  "https://example.com/audio.m4a",
			want: "audio/mp4",
		},
		{
			name: "AAC file",
			url:  "https://example.com/audio.aac",
			want: "audio/aac",
		},
		{
			name: "FLAC file",
			url:  "https://example.com/audio.flac",
			want: "audio/flac",
		},
		{
			name: "WMA file",
			url:  "https://example.com/audio.wma",
			want: "audio/x-ms-wma",
		},
		{
			name: "Opus file",
			url:  "https://example.com/audio.opus",
			want: "audio/opus",
		},
		{
			name: "unknown audio type",
			url:  "https://example.com/audio.unknown",
			want: "",
		},
		{
			name: "non-audio URL",
			url:  "https://example.com/page.html",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectAudioType(tt.url); got != tt.want {
				t.Errorf("DetectAudioType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkIsVideoURL benchmarks video URL detection
func BenchmarkIsVideoURL(b *testing.B) {
	urls := []string{
		"https://example.com/video.mp4",
		"https://www.youtube.com/embed/123456",
		"https://example.com/page.html",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			IsVideoURL(url)
		}
	}
}

// BenchmarkDetectVideoType benchmarks video type detection
func BenchmarkDetectVideoType(b *testing.B) {
	url := "https://example.com/video.mp4"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectVideoType(url)
	}
}

// BenchmarkDetectAudioType benchmarks audio type detection
func BenchmarkDetectAudioType(b *testing.B) {
	url := "https://example.com/audio.mp3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectAudioType(url)
	}
}
