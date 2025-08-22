package dl

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Sentinel error tipleri
var (
	ErrNoDownloader = errors.New("youtube-dl veya yt-dlp bulunamadı")
	ErrDirCreate    = errors.New("klasör oluşturulamadı")
)

// Downloader struct
type Downloader struct {
	BinPath string
	BaseDir string
}

// NewDownloader -> Downloader oluşturur, gerekli binary ve klasörleri kontrol eder
func NewDownloader(baseDir string) (*Downloader, error) {
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		bin, err = exec.LookPath("youtube-dl")
		if err != nil {
			return nil, ErrNoDownloader
		}
	}

	// Klasörü oluştur
	err = os.MkdirAll(baseDir, 0o755)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDirCreate, err)
	}

	return &Downloader{BinPath: bin, BaseDir: baseDir}, nil
}

// Download -> anime adı + bölüm + url alır, dosyayı indirir
func (d *Downloader) Download(source, animeName, url string, episodeNumber, seasonNumber int) error {
	// Çıkış klasörü: ~/Videos/anitr-cli/source/animeName/SxxExx.ext
	// örnek: ~/Videos/anitr-cli/openanime/Bungo Stray Dogs/S04E09.mp4
	outDir := filepath.Join(d.BaseDir, source, animeName)
	err := os.MkdirAll(outDir, 0o755)
	if err != nil {
		return fmt.Errorf("klasör oluşturulamadı: %w", err)
	}

	// Dosya adı formatı: AnimeName-Episode.mp4
	outFile := filepath.Join(outDir, fmt.Sprintf("S%02dE%02d.%%(ext)s", seasonNumber, episodeNumber))

	// Komutu çalıştır
	cmd := exec.Command(d.BinPath, "-o", outFile, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
