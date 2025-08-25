package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/axrona/anitr-cli/internal/player"
)

// AnimeHistoryEntry, her anime için tutulacak bilgiler
type AnimeHistoryEntry struct {
	LastEpisodeIdx  *int       `json:"lastEpisodeIdx"`
	LastEpisodeName string     `json:"lastEpisodeName"`
	AnimeId         *string    `json:"animeId"`
	LastWatched     *time.Time `json:"lastWatched"`
}

// AnimeHistory, source -> anime adı -> struct
type AnimeHistory map[string]map[string]AnimeHistoryEntry

// getHistoryPath cross-platform olarak history.json yolunu döndürür
func getHistoryPath() (string, error) {
	var historyDir string
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("USERPROFILE")
			if appData == "" {
				return "", fmt.Errorf("APPDATA ve USERPROFILE bulunamadı")
			}
		}
		historyDir = filepath.Join(appData, "anitr-cli")
	} else {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME bulunamadı")
		}
		historyDir = filepath.Join(home, ".anitr-cli")
	}

	// Klasör yoksa oluştur
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return "", fmt.Errorf("history klasörü oluşturulamadı: %w", err)
	}

	return filepath.Join(historyDir, "history.json"), nil
}

// ReadAnimeHistory history.json'u okur, yoksa yeni oluşturur
func ReadAnimeHistory() (AnimeHistory, error) {
	path, err := getHistoryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(AnimeHistory), nil
		}
		return nil, fmt.Errorf("history okunamadı: %w", err)
	}

	var history AnimeHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("history parse edilemedi: %w", err)
	}
	return history, nil
}

// WriteAnimeHistory history.json'u yazar
func WriteAnimeHistory(history AnimeHistory) error {
	path, err := getHistoryPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("history serialize edilemedi: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("history yazılamadı: %w", err)
	}
	return nil
}

// UpdateAnimeHistory, mevcut MPV oturumu sırasında animeyi history.json'a kaydeder
func UpdateAnimeHistory(socketPath, source, animeName, episodeName, animeId string, episodeIndex int, logger *Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	updated := false
	for range ticker.C {
		if !player.IsMPVRunning(socketPath) {
			break
		}

		durationVal, err1 := player.MPVSendCommand(socketPath, []interface{}{"get_property", "duration"})
		timePosVal, err2 := player.MPVSendCommand(socketPath, []interface{}{"get_property", "time-pos"})
		if err1 != nil || err2 != nil {
			continue
		}

		duration, ok1 := durationVal.(float64)
		progress, ok2 := timePosVal.(float64)
		if !ok1 || !ok2 {
			continue
		}

		if updated {
			continue
		}

		if progress >= duration-300 { // son 5 dakika
			history, err := ReadAnimeHistory()
			if err != nil {
				logger.LogError(err)
				continue
			}

			sourceEntry, ok := history[source]
			if !ok {
				sourceEntry = make(map[string]AnimeHistoryEntry)
			}

			animeEntry, ok := sourceEntry[animeName]
			if !ok {
				animeEntry = AnimeHistoryEntry{}
			}

			time := time.Now()

			animeEntry = AnimeHistoryEntry{
				LastEpisodeIdx:  &episodeIndex,
				LastEpisodeName: episodeName,
				AnimeId:         &animeId,
				LastWatched:     &time,
			}
			sourceEntry[animeName] = animeEntry
			history[source] = sourceEntry

			if err := WriteAnimeHistory(history); err != nil {
				logger.LogError(err)
				continue
			}
			updated = true
		}
	}
}
