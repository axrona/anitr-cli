package player

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/axrona/anitr-cli/internal/ipc"
)

// MPVParams yapısı, MPV oynatıcı parametrelerini tutar.
type MPVParams struct {
	Url         string  // Oynatılacak video URL'si
	SubtitleUrl *string // Altyazı URL'si (isteğe bağlı)
	Title       string  // Video başlığı
}

// isMPVInstalled fonksiyonu, sistemde MPV oynatıcısının yüklü olup olmadığını kontrol eder.
func isMPVInstalled() error {
	mpvBinary := "mpv"
	if runtime.GOOS == "windows" {
		mpvBinary = "mpv.exe"
	}
	_, err := exec.LookPath(mpvBinary) // MPV'nin sistemde bulunup bulunmadığını kontrol eder
	return err
}

// getMPVSocketPath platforma göre IPC yolunu döner
func getMPVSocketPath() string {
	if runtime.GOOS == "windows" {
		// Windows named pipe yolu
		return `\\.\pipe\anitr-cli-410`
	}
	// Linux/macOS unix socket
	return filepath.Join(os.TempDir(), "anitr-cli-410.sock")
}

// getMPVBinary platforma göre mpv binary adını döner
func getMPVBinary() string {
	if runtime.GOOS == "windows" {
		return "mpv.exe"
	}
	return "mpv"
}

// Play fonksiyonu, verilen parametrelerle MPV oynatıcıyı başlatır.
func Play(params MPVParams) (*exec.Cmd, string, error) {
	mpvSocketPath := getMPVSocketPath()

	// MPV'nin yüklü olup olmadığını kontrol et
	if err := isMPVInstalled(); err != nil {
		return nil, "", errors.New("mpv sisteminizde yüklü değil") // Yükleme hatası
	}

	// MPV başlatma komutunu oluştur
	args := []string{
		"--fullscreen", // Tam ekran başlat
		"--save-position-on-quit",
		fmt.Sprintf("--title=%s", params.Title),
		fmt.Sprintf("--force-media-title=%s", params.Title),
		"--idle=once", "--really-quiet", "--no-terminal",
		fmt.Sprintf("--input-ipc-server=%s", mpvSocketPath),
	}

	// Platform bazlı user-agent ve referrer ayarı (isteğe bağlı)
	if runtime.GOOS == "linux" {
		args = append(args,
			"--user-agent=Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/137.0.0.0 Safari/537.36",
			"--referrer=https://yeshi.eu.org/")
	} else if runtime.GOOS == "windows" {
		args = append(args,
			"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			"--referrer=https://yeshi.eu.org/")
	}

	// Eğer altyazı URL'si varsa, altyazı dosyasını ekle
	if params.SubtitleUrl != nil && *params.SubtitleUrl != "" {
		args = append(args, "--sub-file", *params.SubtitleUrl)
	}

	// Video URL'sini ekle
	args = append(args, params.Url)

	mpvBinary := getMPVBinary()
	cmd := exec.Command(mpvBinary, args...)
	if err := cmd.Start(); err != nil {
		return cmd, "", err // Başlatma hatası
	}

	// MPV'nin IPC soketinin hazır olmasını bekle
	maxRetries := 25
	retryDelay := 300 * time.Millisecond
	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryDelay)
		conn, err := ipc.ConnectToPipe(mpvSocketPath)
		if err == nil {
			conn.Close()
			return cmd, mpvSocketPath, nil
		}
	}

	return cmd, "", errors.New("MPV socket hazır değil, başlatılamadı")
}

// MPVSendCommand, MPV'ye bir komut gönderir ve yanıtını döner.
func MPVSendCommand(ipcSocketPath string, command []interface{}) (interface{}, error) {
	var lastErr error
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	// Komutun gönderilmesini birkaç kez dener
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		conn, err := ipc.ConnectToPipe(ipcSocketPath)
		if err != nil {
			lastErr = err
			continue
		}
		defer func() { _ = conn.Close() }()

		// Komutu JSON formatına dönüştür
		commandStr, err := json.Marshal(map[string]interface{}{
			"command": command,
		})
		if err != nil {
			return nil, err
		}

		// Komutu sokete gönder
		_, err = conn.Write(append(commandStr, '\n'))
		if err != nil {
			lastErr = err
			continue
		}

		// Yanıtı oku
		buf := make([]byte, 4096)
		if deadline, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
			deadline.SetReadDeadline(time.Now().Add(1 * time.Second))
		}

		n, err := conn.Read(buf)
		if err != nil {
			lastErr = err
			continue
		}

		// Yanıtı çözümle
		var response map[string]interface{}
		if err := json.Unmarshal(buf[:n], &response); err != nil {
			lastErr = err
			continue
		}

		// Yanıt verisi varsa, döndür
		if data, exists := response["data"]; exists {
			return data, nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("command failed after %d attempts: %w", maxRetries, lastErr) // Komut hatası
}

// SeekMPV, MPV'yi belirli bir zaman noktasına kaydırır.
func SeekMPV(ipcSocketPath string, time int) (interface{}, error) {
	command := []interface{}{"seek", time, "absolute"} // Mutlak zaman noktasına kaydır
	return MPVSendCommand(ipcSocketPath, command)
}

// GetMPVPausedStatus, MPV'nin duraklatılıp duraklatılmadığını kontrol eder.
func GetMPVPausedStatus(ipcSocketPath string) (bool, error) {
	status, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "pause"})
	if err != nil || status == nil {
		return false, err // Durum hatası
	}

	paused, ok := status.(bool)
	if ok {
		return paused, nil // Duraklatma durumu
	}
	return false, nil
}

// GetMPVPlaybackSpeed, MPV oynatma hızını döner.
func GetMPVPlaybackSpeed(ipcSocketPath string) (float64, error) {
	speed, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "speed"})
	if err != nil || speed == nil {
		return 0, err // Hız hatası
	}

	currentSpeed, ok := speed.(float64)
	if ok {
		return currentSpeed, nil // Oynatma hızı
	}

	return 0, nil
}

// GetPercentageWatched, izlenen kısmın yüzdesini döner.
func GetPercentageWatched(ipcSocketPath string) (float64, error) {
	currentTime, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "time-pos"})
	if err != nil || currentTime == nil {
		return 0, err // Zaman hatası
	}

	duration, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "duration"})
	if err != nil || duration == nil {
		return 0, err // Süre hatası
	}

	currTime, ok1 := currentTime.(float64)
	dur, ok2 := duration.(float64)

	if ok1 && ok2 && dur > 0 {
		percentageWatched := (currTime / dur) * 100 // İzlenen yüzdesi
		return percentageWatched, nil
	}

	return 0, nil
}

// PercentageWatched, verilen oynatma zamanı ve toplam süreye göre izlenen yüzdesi hesaplar.
func PercentageWatched(playbackTime int, duration int) float64 {
	if duration > 0 {
		percentage := (float64(playbackTime) / float64(duration)) * 100
		return percentage
	}
	return float64(0)
}

// HasActivePlayback, MPV'nin aktif bir oynatma durumu olup olmadığını kontrol eder.
func HasActivePlayback(ipcSocketPath string) (bool, error) {
	maxRetries := 3
	var lastErr error

	// Aktif oynatma durumu kontrolü birkaç kez denenir
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(200 * time.Millisecond)
		}

		timePos, err := MPVSendCommand(ipcSocketPath, []interface{}{"get_property", "time-pos"})
		if err != nil {
			if strings.Contains(err.Error(), "property unavailable") {
				return false, nil // Özellik mevcut değil
			}

			if strings.Contains(err.Error(), "connect: connection refused") ||
				strings.Contains(err.Error(), "connect: no such file or directory") {
				lastErr = err
				continue
			}

			return false, fmt.Errorf("error getting time-pos: %w", err) // Zaman alınamadı
		}

		if timePos != nil {
			return true, nil // Oynatma aktif
		}

		return false, nil
	}

	return false, fmt.Errorf("failed to check playback status: %w", lastErr) // Durum kontrolü başarısız
}

// IsMPVRunning, MPV'nin çalışıp çalışmadığını kontrol eder.
func IsMPVRunning(socketPath string) bool {
	if socketPath == "" {
		return false // Geçersiz soket yolu
	}

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(200 * time.Millisecond)
		}

		conn, err := ipc.ConnectToPipe(socketPath)
		if err != nil {
			continue // Bağlantı hatası
		}
		defer conn.Close()

		_, err = MPVSendCommand(socketPath, []interface{}{"get_property", "pid"})
		if err == nil {
			return true // MPV çalışıyor
		}

	}

	return false // MPV çalışmıyor
}
