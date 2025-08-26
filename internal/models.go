// Package internal, uygulama genelinde kullanılan ortak yapı ve yardımcı işlevleri içerir.
package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config, uygulamanın temel yapılandırma ayarlarını temsil eder.
type Config struct {
	BaseUrl        string            // API'nin temel adresi
	AlternativeUrl string            // Alternatif API adresi (fallback)
	VideoPlayers   []string          // Kullanılabilir video oynatıcılar
	HttpHeaders    map[string]string // HTTP isteklerinde kullanılacak başlıklar
}

// UiParams, UI (kullanıcı arayüzü) ile ilgili parametreleri temsil eder.
type UiParams struct {
	Mode      string    // Arayüz modu: "rofi" veya "tui"
	List      *[]string // Liste halinde kullanıcıya gösterilecek seçenekler
	Label     string    // UI öğesi için başlık/etiket
	RofiFlags *string   // Rofi'ye özel ek parametreler (varsa)
}

// RPCParams, Discord Rich Presence için gönderilecek bilgileri içerir.
type RPCParams struct {
	Type       int       // Playing / Listening / Watching / Competing
	Details    string    // Aktivite detayı
	State      string    // Kullanıcı durumu
	LargeImage string    // Büyük görselin adı
	LargeText  string    // Büyük görsel üzerine gelindiğinde gösterilecek yazı
	SmallImage string    // Küçük görselin adı
	SmallText  string    // Küçük görsel üzerine gelindiğinde gösterilecek yazı
	Timestamp  time.Time // Timestamp
}

// GetStringPtr, map içinden verilen anahtara karşılık gelen değeri *string olarak döner.
// Değer string değilse veya bulunamazsa nil döner.
func GetStringPtr(m map[string]interface{}, key string) *string {
	if val, ok := m[key].(string); ok {
		return &val
	}
	return nil
}

// GetString, map içinden verilen anahtara karşılık gelen string değeri döner.
// Değer string değilse veya bulunamazsa boş string ("") döner.
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetJson, verilen URL'ye HTTP GET isteği gönderir, gelen JSON yanıtı çözümler.
// Başarılı olursa çözülmüş veriyi interface{} olarak döner, aksi hâlde hata döner.
func GetJson(url string, headers map[string]string) (interface{}, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTP isteği oluşturulamadı: %w", err)
	}

	// İstek başlıklarını ayarla
	for k, m := range headers {
		req.Header.Set(k, m)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP isteği başarısız: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTTP yanıtı okunamadı: %w", err)
	}

	var result interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("JSON ayrıştırma başarısız: %w", err)
	}

	return result, nil
}
