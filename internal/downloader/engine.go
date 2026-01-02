package downloader

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"ytdl-server/internal/models"

	"github.com/kkdai/youtube/v2"
)

type ProgressCallback func(percentage float64)

// DOWNLOAD AND MUTEX
func Process(job *models.Job, quality string, tempDir string, onProgress ProgressCallback) error {
	client := youtube.Client{}
	video, err := client.GetVideo(job.VideoID)
	if err != nil {
		return fmt.Errorf("video info error: %v", wrapError(err))
	}

	targetHeight := parseQuality(quality)
	videoFormat := findBestVideoFormat(video.Formats, targetHeight)
	audioFormat := findBestAudioFormat(video.Formats)

	if videoFormat == nil || audioFormat == nil {
		return fmt.Errorf("format not found")
	}

	// FILE PATHS
	safeTitle := sanitizeFilename(video.Title)
	job.Filename = fmt.Sprintf("%s_%dp.mp4", safeTitle, targetHeight)

	videoTemp := filepath.Join(tempDir, fmt.Sprintf("v_%s.mp4", job.ID))
	audioTemp := filepath.Join(tempDir, fmt.Sprintf("a_%s.m4a", job.ID))

	totalSize := videoFormat.ContentLength + audioFormat.ContentLength
	var currentBytes int64 = 0
	var mu sync.Mutex

	track := func(n int) {
		mu.Lock()
		defer mu.Unlock()
		currentBytes += int64(n)
		// engine.go içinde yüzde hesaplanan yer
		if totalSize > 0 {
    		pct := float64(currentBytes) / float64(totalSize) * 100
    		if pct > 99.9 {
        	pct = 99.9
    	}
    	onProgress(pct) 
		
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var errV, errA error

	go func() {
		defer wg.Done()
		errV = downloadStream(client, video, videoFormat, videoTemp, track)
	}()
	go func() {
		defer wg.Done()
		errA = downloadStream(client, video, audioFormat, audioTemp, track)
	}()
	wg.Wait()

	if errV != nil {
		return errV
	}
	if errA != nil {
		return errA
	}

	// Muxing
	onProgress(99.9) // Muxing başlıyor
	cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-i", videoTemp, "-i", audioTemp, "-c", "copy", job.FilePath)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %s", string(out))
	}

	os.Remove(videoTemp)
	os.Remove(audioTemp)

	// 0 Byte kontrolü
	if info, err := os.Stat(job.FilePath); err != nil || info.Size() == 0 {
		return fmt.Errorf("generated file is empty")
	}

	return nil
}

// --- Helpers (Private) ---

func downloadStream(c youtube.Client, v *youtube.Video, f *youtube.Format, path string, cb func(int)) error {
	stream, _, err := c.GetStream(v, f)
	if err != nil {
		return err
	}
	defer stream.Close()

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, 32*1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			file.Write(buf[:n])
			cb(n)
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func parseQuality(q string) int {
	if q == "4k" {
		return 2160
	}
	digits := ""
	for _, c := range q {
		if c >= '0' && c <= '9' {
			digits += string(c)
		}
	}
	if digits == "" {
		return 0
	}
	val, _ := strconv.Atoi(digits)
	return val
}

func sanitizeFilename(name string) string {
	safe := strings.ReplaceAll(name, " ", "_")
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return -1
		}
		return r
	}, safe)
}

func findBestVideoFormat(formats youtube.FormatList, targetHeight int) *youtube.Format {
	var best *youtube.Format
	for _, f := range formats {
		if strings.Contains(f.MimeType, "video") {
			h := parseQuality(f.QualityLabel)
			if h == targetHeight {
				return &f
			}
			if best == nil || (h > parseQuality(best.QualityLabel) && h <= targetHeight) {
				temp := f
				best = &temp
			}
		}
	}
	if best == nil {
		for _, f := range formats {
			if strings.Contains(f.MimeType, "video") {
				if best == nil || parseQuality(f.QualityLabel) > parseQuality(best.QualityLabel) {
					temp := f
					best = &temp
				}
			}
		}
	}
	return best
}

func findBestAudioFormat(formats youtube.FormatList) *youtube.Format {
	var best *youtube.Format
	for _, f := range formats {
		if strings.Contains(f.MimeType, "audio") {
			if best == nil || (strings.Contains(f.MimeType, "mp4") && !strings.Contains(best.MimeType, "mp4")) {
				temp := f
				best = &temp
			}
		}
	}
	return best
}

func wrapError(err error) string {
    msg := err.Error()
    switch {
    case strings.Contains(msg, "permission denied"):
        return "Storage permission denied. Please contact system administrator."
    case strings.Contains(msg, "no space left"):
        return "Disk space exhausted. Cannot complete download."
    case strings.Contains(msg, "ffmpeg"):
        return "Media processing error (FFmpeg failed). Please try again."
    case strings.Contains(msg, "cipher") || strings.Contains(msg, "signature"):
        return "YouTube restricted access to this video (Cipher/Signature error)."
    case strings.Contains(msg, "403"):
        return "Access forbidden. YouTube might be throttling the server IP."
    default:
        // Genel teknik hata mesajı (Path ifşasını önler)
        return "An unexpected technical error occurred during processing."
    }
}