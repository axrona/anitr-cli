package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	ColorReset = "\033[0m"
	ColorRed   = "\033[31m"
	ColorCyan  = "\033[36m"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// GitHub API'den JSON verisi çeker
func fetchAPI(url string) (*githubRelease, error) {
	if url == "" {
		return nil, errors.New("API URL boş")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API'ye erişim başarısız: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API hatası: HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("API'den veri okunamadı: %w", err)
	}

	var release githubRelease
	if err = json.Unmarshal(respBody, &release); err != nil {
		return nil, fmt.Errorf("API verisi ayrıştırılamadı: %w", err)
	}

	if release.TagName == "" {
		return nil, errors.New("API yanıtında tag_name bulunamadı")
	}

	return &release, nil
}

// Sürüm kontrolü yapar, yeni bir güncelleme olup olmadığını döner
func FetchUpdates() (string, error) {
	release, err := fetchAPI(githubAPI)
	if err != nil {
		return "", fmt.Errorf("güncelleme verileri alınamadı: %w", err)
	}

	currentVer, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("geçerli sürüm numarası geçersiz: %v", err)
	}

	latestVer, err := semver.NewVersion(release.TagName)
	if err != nil {
		return "", fmt.Errorf("en son sürüm numarası geçersiz: %v", err)
	}

	if !currentVer.LessThan(latestVer) {
		return "Zaten en son sürümdesiniz.", nil
	}
	return fmt.Sprintf("Yeni sürüm bulundu: %s -> %s", version, release.TagName), nil
}

// Sürüm bilgisini döner
func Version() string {
	return fmt.Sprintf(
		"anitr-cli %s\nLisans: GPL 3.0 (Özgür Yazılım)\nDestek ver: %s\n\nGo sürümü: %s\n",
		version, repoLink, buildEnv,
	)
}

// Güncellemeleri kontrol eder ve varsa kullanıcıya bildirir
func CheckUpdates() {
	msg, err := FetchUpdates()
	if err != nil {
		fmt.Println(ColorRed + "Güncelleme kontrolü sırasında bir hata oluştu: " + err.Error() + ColorReset)
		time.Sleep(2 * time.Second)
		return
	}

	if msg != "Zaten en son sürümdesiniz." {
		fmt.Println(ColorCyan + msg + ColorReset)
		time.Sleep(2 * time.Second)
	}
}
