package utils

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xeyossr/anitr-cli/internal"
	"github.com/xeyossr/anitr-cli/internal/ui"
)

var ErrQuit = errors.New("quit requested")

// Logger, hata ve mesajları bir dosyaya yazmak için yapılandırılmış bir log yapısıdır.
type Logger struct {
	File *os.File    // Log dosyasının kendisi
	Log  *log.Logger // Log işlemini gerçekleştiren nesne
}

// getTempDir işletim sistemine göre geçici dizin döner.
func getTempDir() string {
	if runtime.GOOS == "windows" {
		// Windows ortam değişkeni TEMP veya TMP
		if temp := os.Getenv("TEMP"); temp != "" {
			return temp
		}
		if tmp := os.Getenv("TMP"); tmp != "" {
			return tmp
		}
		// Fallback
		return `C:\Temp`
	}
	// Unix benzeri sistemler için /tmp
	return "/tmp"
}

// GetImage, verilen URL'den bir görsel indirir ve geçici bir dosyaya kaydeder.
func GetImage(url string) (string, error) {
	tempDir := getTempDir()
	tempPath := filepath.Join(tempDir, "poster.png")

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("görsel indirilemedi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("görsel isteğine başarısız yanıt: %s", resp.Status)
	}

	out, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("geçici dosya oluşturulamadı: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("görsel yazılamadı: %w", err)
	}

	return tempPath, nil
}

// NewLogger, işletim sistemine göre uygun dizinde bir log dosyası oluşturur ve Logger döner.
func NewLogger() (*Logger, error) {
	tempDir := getTempDir()
	logPath := filepath.Join(tempDir, "anitr-cli.log")

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("log dosyası açılamadı: %w", err)
	}

	logger := log.New(file, "", log.LstdFlags|log.Lmsgprefix)

	return &Logger{
		File: file,
		Log:  logger,
	}, nil
}

// ... diğer fonksiyonlar değişmeden kalabilir ...

// LogError, hata objesini loglar (nil ise işlem yapılmaz).
func (l *Logger) LogError(err error) {
	if err == nil {
		return
	}

	l.Log.Printf("[ERROR] %v\n", err)
}

// LogMsg, belirtilen formatta bir log mesajı yazar.
func (l *Logger) LogMsg(format string, a ...interface{}) {
	l.Log.Printf(format, a...)
}

// Close, log dosyasını kapatır.
func (l *Logger) Close() error {
	return l.File.Close()
}

// FailIfErr, kritik hatalarda loglar, kullanıcıyı bilgilendirir ve uygulamayı kapatır.
func FailIfErr(params internal.UiParams, err error, logger *Logger) {
	if err != nil {
		if errors.Is(err, ErrQuit) {
			os.Exit(0)
		}

		logger.LogError(err)
		ui.ShowError(params, fmt.Sprint(err))
		logger.Close()
		os.Exit(1)
	}
}

// CheckErr, hata varsa loglar, ekranda gösterir ve kullanıcıdan devam için giriş bekler.
func CheckErr(params internal.UiParams, err error, logger *Logger) bool {
	if err != nil {
		if errors.Is(err, ErrQuit) {
			os.Exit(0)
		}

		logger.LogError(err)
		ui.ShowError(params, fmt.Sprint(err))
		fmt.Scanln()
		return false
	}
	return true
}

// IsValidImage, verilen URL'nin geçerli bir görsel olup olmadığını kontrol eder.
func IsValidImage(url string) bool {
	client := &http.Client{}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	return resp.StatusCode == 200 && strings.HasPrefix(contentType, "image/")
}

// NormalizeTurkishToASCII, Türkçe karakterleri ASCII eşdeğerleri ile değiştirir.
func NormalizeTurkishToASCII(input string) string {
	replacer := strings.NewReplacer(
		"ö", "o", "ü", "u", "ı", "i", "ç", "c", "ş", "s", "ğ", "g",
		"Ö", "O", "Ü", "U", "İ", "I", "Ç", "C", "Ş", "S", "Ğ", "G",
	)
	return replacer.Replace(input)
}

// Ptr, verilen değerin pointer'ını döner (herhangi bir tip için).
func Ptr[T any](val T) *T {
	return &val
}

func VideosDir() string {
	if runtime.GOOS == "windows" {
		// Kullanıcı profil dizini → %USERPROFILE%
		userProfile := os.Getenv("USERPROFILE")
		if userProfile == "" {
			userProfile = "." // fallback
		}
		return filepath.Join(userProfile, "Videos")
	} else {
		// Linux / Mac
		home := os.Getenv("HOME")
		if home == "" {
			home = "."
		}
		return filepath.Join(home, "Videos")
	}
}
