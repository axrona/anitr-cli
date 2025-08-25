package animecix

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/xeyossr/anitr-cli/internal"
	"github.com/xeyossr/anitr-cli/internal/models"
	"github.com/xeyossr/anitr-cli/internal/utils"
)

type AnimeCix struct{}

// AnimeCix API için yapılandırma ayarları
var configAnimecix = internal.Config{
	BaseUrl:        "https://animecix.tv/",
	AlternativeUrl: "https://mangacix.net/",
	VideoPlayers:   []string{"tau-video.xyz", "sibnet"},
	HttpHeaders:    map[string]string{"Accept": "application/json", "User-Agent": "Mozilla/5.0", "x-e-h": "=.a"},
}

// VideoURL, video URL'sinin etiket ve bağlantısını tutar
type VideoURL struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// VideoResponse, video URL'leri için gelen yanıtın yapısı
type VideoResponse struct {
	URLs []VideoURL `json:"urls"`
}

// Source, AnimeCix kaynağının adını döner
func (a AnimeCix) Source() string {
	return "AnimeciX"
}

// GetSearchData, verilen sorguya göre anime verilerini döner
func (a AnimeCix) GetSearchData(query string) ([]models.Anime, error) {
	// Türkçe karakterleri ASCII'ye dönüştür ve boşlukları "-" ile değiştir
	normalizedQuery := utils.NormalizeTurkishToASCII(query)
	normalizedQuery = strings.ReplaceAll(normalizedQuery, " ", "-")

	// Anime arama verilerini al
	data, err := FetchAnimeSearchData(normalizedQuery)
	if err != nil {
		return nil, err
	}

	var returnData []models.Anime

	// Alınan verileri Anime modeline dönüştür
	for _, item := range data {
		id, ok := item["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("id verisi beklenen formatta değil")
		}
		intId := int(id)
		title, ok := item["name"].(string)
		if !ok {
			return nil, fmt.Errorf("title verisi beklenen formatta değil")
		}

		animeType, ok := item["type"].(string)
		if !ok {
			animeType = ""
		}

		titleType, ok := item["title_type"].(string)
		if !ok {
			titleType = ""
		}

		poster, ok := item["poster"].(string)
		if !ok {
			poster = ""
		}

		// Anime bilgilerini ekle
		returnData = append(returnData, models.Anime{
			ID:        &intId,
			Title:     title,
			Type:      &animeType,
			TitleType: &titleType,
			ImageURL:  poster,
		})
	}

	return returnData, nil
}

// GetAnimeByID, verilen ID'ye göre tek anime verisini döner
func (a AnimeCix) GetAnimeByID(idstr string) (*models.Anime, error) {
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%ssecure/titles/%d?titleId=%d", configAnimecix.BaseUrl, id, id)
	data, err := internal.GetJson(url, configAnimecix.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("anime verisi alınamadı: %w", err)
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data verisi beklenen formatta değil")
	}

	titleMap, ok := dataMap["title"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'title' verisi yok veya beklenen formatta değil")
	}

	name, _ := titleMap["name"].(string)
	animeType, _ := titleMap["type"].(string)
	titleType, _ := titleMap["title_type"].(string)
	poster, _ := titleMap["poster"].(string)

	return &models.Anime{
		ID:        &id,
		Title:     name,
		Type:      &animeType,
		TitleType: &titleType,
		ImageURL:  poster,
	}, nil
}

// GetSeasonsData, anime için sezon bilgilerini döner
func (a AnimeCix) GetSeasonsData(params models.SeasonParams) ([]models.Season, error) {
	// Sezon verilerini al
	data, err := FetchAnimeSeasonsData(*params.Id)
	if err != nil {
		return nil, err
	}

	// Alınan verileri sezon yapısına dönüştür
	return []models.Season{
		{
			Seasons: &data,
		},
	}, nil
}

// GetEpisodesData, sezon için bölüm bilgilerini döner
func (a AnimeCix) GetEpisodesData(params models.EpisodeParams) ([]models.Episode, error) {
	// Bölüm verilerini al
	episodesRaw, err := FetchAnimeEpisodesData(*params.SeasonID)
	if err != nil {
		return nil, fmt.Errorf("bölüm verileri alınamadı: %w", err)
	}

	var episodes []models.Episode
	// Bölümleri modele dönüştür
	for i, item := range episodesRaw {
		title, _ := item["name"].(string)
		url, _ := item["url"].(string)
		episode := models.Episode{
			ID:     url,
			Title:  title,
			Number: i + 1,
			Extra:  map[string]interface{}{"season_num": item["season_num"]},
		}
		episodes = append(episodes, episode)
	}

	return episodes, nil
}

// GetWatchData, anime için izleme verilerini döner
func (a AnimeCix) GetWatchData(req models.WatchParams) ([]models.Watch, error) {
	// Verilerin eksik olup olmadığını kontrol et
	if req.IsMovie == nil || req.Url == nil || req.Id == nil || req.Extra == nil {
		return nil, fmt.Errorf("panic")
	}

	// Parametreleri al
	var (
		isMovie      bool                   = *req.IsMovie
		url          string                 = *req.Url
		id           int                    = *req.Id
		Extra        map[string]interface{} = *req.Extra
		seasonIndex  int                    = Extra["seasonIndex"].(int)
		episodeIndex int                    = Extra["episodeIndex"].(int)
	)

	// Eğer filmse, film izleme verilerini al
	if isMovie {
		data, err := AnimeMovieWatchApiUrl(id)
		if err != nil {
			return nil, fmt.Errorf("film verileri alınamadı: %w", err)
		}

		// Video akışlarını kontrol et
		streams, ok := data["video_streams"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("video_streams verisi beklenen formatta değil")
		}

		var labels []string
		var urls []string
		// Her bir video akışını listele
		for _, s := range streams {
			item, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			label, _ := item["label"].(string)
			url, _ := item["url"].(string)

			labels = append(labels, label)
			urls = append(urls, url)
		}

		var captionUrl *string
		if c, ok := data["caption_url"].(string); ok {
			captionUrl = &c
		}

		// İzleme verilerini döndür
		watch := models.Watch{
			Labels:    labels,
			Urls:      urls,
			TRCaption: captionUrl,
		}

		return []models.Watch{watch}, nil
	}

	// Bölüm izleme verilerini al
	videoStreams, err := AnimeWatchApiUrl(url)
	if err != nil {
		return nil, fmt.Errorf("bölüm verileri alınamadı: %w", err)
	}

	// Altyazıyı al
	captionUrl, err := FetchTRCaption(seasonIndex, episodeIndex, id)
	if err != nil {
		captionUrl = ""
	}

	// Video akışlarını listele
	var labels []string
	var urls []string
	for _, entry := range videoStreams {
		labels = append(labels, entry["label"])
		urls = append(urls, entry["url"])
	}

	// İzleme verisini döndür
	watch := models.Watch{
		Labels:    labels,
		Urls:      urls,
		TRCaption: &captionUrl,
	}

	return []models.Watch{watch}, nil
}

// FetchAnimeSearchData, anime arama verilerini alır
func FetchAnimeSearchData(query string) ([]map[string]interface{}, error) {
	// Arama URL'sini oluştur
	url := fmt.Sprintf("%ssecure/search/%s?type=&limit=20", configAnimecix.BaseUrl, query)
	// JSON verisini al
	data, err := internal.GetJson(url, configAnimecix.HttpHeaders)
	if err != nil {
		return nil, err
	}

	// Veriyi işleyerek gerekli alanları çıkart
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data verisi beklenen formatta değil")
	}

	resultsRaw, exists := m["results"]
	if !exists {
		return nil, fmt.Errorf("'results' verisi bulunamadı")
	}

	resultsSlice, ok := resultsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'results' verisi beklenen formatta değil")
	}

	var parsed []map[string]interface{}
	// Her bir sonucu işleyip döndür
	for _, item := range resultsSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("'item' verisi beklenen formatta değil")
		}

		entry := map[string]interface{}{
			"name":       itemMap["name"],
			"id":         itemMap["id"],
			"type":       itemMap["type"],
			"title_type": itemMap["title_type"],
			"poster":     itemMap["poster"],
		}

		parsed = append(parsed, entry)
	}

	return parsed, nil
}

// FetchAnimeSeasonsData, anime için sezon verilerini alır
func FetchAnimeSeasonsData(id int) ([]int, error) {
	url := fmt.Sprintf("%ssecure/related-videos?episode=1&season=1&titleId=%d&videoId=637113", configAnimecix.AlternativeUrl, id)
	data, err := internal.GetJson(url, configAnimecix.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("sezon verileri alınamadı: %w", err)
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data beklenen formatta değil")
	}

	videosField, ok := dataMap["videos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("'videos' verisi yok veya beklenen formatta değil")
	}

	video, ok := videosField[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'videos'[0] verisi beklenen formatta değil")
	}

	title, ok := video["title"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'title' verisi yok veya beklenen formatta değil")
	}

	seasons, ok := title["seasons"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("'seasons' verisi yok veya beklenen formatta değil")
	}

	count := len(seasons)
	indices := make([]int, count)
	for i := range indices {
		indices[i] = i
	}

	return indices, nil
}

// FetchAnimeEpisodesData, anime için bölüm verilerini alır
func FetchAnimeEpisodesData(id int) ([]map[string]interface{}, error) {
	var episodes []map[string]interface{}
	seenEpisodes := make(map[string]bool)
	seasons, err := FetchAnimeSeasonsData(id)
	if err != nil {
		return nil, fmt.Errorf("sezon verileri alınamadı: %w", err)
	}

	// Her sezon için bölüm verilerini al
	for _, seasonIndex := range seasons {
		url := fmt.Sprintf("%ssecure/related-videos?episode=1&season=%d&titleId=%d&videoId=637113", configAnimecix.AlternativeUrl, seasonIndex+1, id)
		data, err := internal.GetJson(url, configAnimecix.HttpHeaders)
		if err != nil {
			return nil, fmt.Errorf("bölüm verileri alınamadı: %w", err)
		}

		dataMap, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("data beklenen formatta değil")
		}

		videosRaw, ok := dataMap["videos"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("'videos' verisi yok veya beklenen formatta değil")
		}

		// Her bir video için bölüm verilerini ekle
		for _, video := range videosRaw {
			video, ok := video.(map[string]interface{})

			if !ok {
				return nil, err
			}

			name, ok := video["name"].(string)
			if !ok {
				return nil, fmt.Errorf("name verisi beklenen formatta değil")
			}

			if !seenEpisodes[name] {
				episodeUrl, ok := video["url"].(string)

				if !ok {
					return nil, fmt.Errorf("url verisi beklenen formatta değil")
				}

				seasonNum := video["season_num"]
				episode := map[string]interface{}{"name": name, "url": episodeUrl, "season_num": seasonNum}
				episodes = append(episodes, episode)
				seenEpisodes[name] = true
			}
		}
	}

	return episodes, nil
}

// AnimeWatchApiUrl, anime için izleme verilerini döner
func AnimeWatchApiUrl(Url string) ([]map[string]string, error) {
	watch_url := fmt.Sprintf("%s%s", configAnimecix.BaseUrl, Url)
	resp, err := http.Get(watch_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 422 hatası alırsak, beklenen formatta veriler yok demektir
	if resp.StatusCode == 422 {
		return nil, errors.New("bölüm verisi beklenen formatta değil")
	}

	// Gelen URL'yi işle ve video verilerine ulaş
	finalUrl := resp.Request.URL.String()
	parsedUrl, err := url.Parse(finalUrl)
	if err != nil {
		return nil, err
	}

	// URL'yi çözümleyip, verileri al
	pathParts := strings.Split(parsedUrl.Path, "/")
	if len(pathParts) < 3 {
		return nil, fmt.Errorf("path verisi beklenen formatta değil")
	}

	embedID := pathParts[2]
	queryParams := parsedUrl.Query()
	vid := queryParams.Get("vid")

	apiUrl := fmt.Sprintf("https://%s/api/video/%s?vid=%s", configAnimecix.VideoPlayers[0], embedID, vid)
	response, err := http.Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("video verileri alınamadı: %w", err)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("video verileri okunamadı: %w", err)
	}

	var videoResp VideoResponse
	err = json.Unmarshal(body, &videoResp)
	if err != nil {
		return nil, fmt.Errorf("video verileri ayrıştırılamadı: %w", err)
	}

	// Video URL'leri ve etiketlerini döndür
	results := []map[string]string{}
	for _, item := range videoResp.URLs {
		entry := map[string]string{
			"label": item.Label,
			"url":   item.URL,
		}

		results = append(results, entry)
	}

	return results, nil
}

// FetchTRCaption, Türkçe altyazıyı döner
func FetchTRCaption(seasonIndex, episodeIndex, id int) (string, error) {
	url := fmt.Sprintf("%ssecure/related-videos?episode=1&season=%d&titleId=%d&videoId=637113", configAnimecix.AlternativeUrl, seasonIndex+1, id)
	data, err := internal.GetJson(url, configAnimecix.HttpHeaders)
	if err != nil {
		return "", fmt.Errorf("altyazı verileri alınamadı: %w", err)
	}

	// Gelen veriyi çözümle ve Türkçe altyazıyı bul
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data verisi beklenen formatta değil")
	}

	videosSlice, ok := dataMap["videos"].([]interface{})
	if !ok {
		return "", fmt.Errorf("'videos' verisi yok veya beklenen formatta değil")
	}

	// İlgili bölümü al
	video, ok := videosSlice[episodeIndex].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("episode verisi yok veya beklenen formatta değil")
	}

	// Altyazıyı kontrol et
	captions, ok := video["captions"].([]interface{})
	if !ok {
		return "", fmt.Errorf("'captions' verisi yok veya beklenen formatta değil")
	}

	for _, caption := range captions {
		caption, ok := caption.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("caption verisi yok veya beklenen formatta değil")
		}

		lang, ok := caption["language"].(string)
		if ok && lang == "tr" {
			return caption["url"].(string), nil
		}
	}

	// Eğer Türkçe altyazı bulunmazsa, bir hata döndür
	if len(captions) == 0 {
		return "", fmt.Errorf("altyazı bulunamadı")
	}
	caption0 := captions[0].(map[string]interface{})
	return caption0["url"].(string), nil
}

// AnimeMovieWatchApiUrl, film için video URL'lerini döner
func AnimeMovieWatchApiUrl(id int) (map[string]interface{}, error) {
	Url := fmt.Sprintf("%ssecure/titles/%d?titleId=%d", configAnimecix.BaseUrl, id, id)

	data, err := internal.GetJson(Url, configAnimecix.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("anime verisi alınamadı: %w", err)
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data beklenen formatta değil")
	}

	// Video verileriyle birlikte altyazıları da döndür
	titleMap, ok := dataMap["title"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'title' verisi yok veya beklenen formatta değil")
	}

	videosRaw, ok := titleMap["videos"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("'videos' verisi yok veya beklenen formatta değil")
	}

	for _, video := range videosRaw {
		video, ok := video.(map[string]interface{})

		if !ok {
			return nil, fmt.Errorf("'videos' verisi yok veya beklenen formatta değil")
		}

		videoUrl, ok := video["url"].(string)

		if !ok {
			return nil, fmt.Errorf("'url' verisi yok veya beklenen formatta değil")
		}

		// Video URL'yi çözümle
		client := &http.Client{}
		req, err := http.NewRequest("GET", videoUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("HTTP isteği oluşturulamadı: %w", err)
		}

		req.Header.Set("Accept", configAnimecix.HttpHeaders["Accept"])
		req.Header.Set("User-Agent", configAnimecix.HttpHeaders["User-Agent"])
		req.Header.Set("x-e-h", "=.a")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("video verileri alınamadı: %w", err)
		}

		resp.Body.Close()

		// Alınan URL'yi işleyerek video verilerini döndür
		finalUrl := resp.Request.URL.String()
		parsedUrl, err := url.Parse(finalUrl)
		if err != nil {
			return nil, fmt.Errorf("URL ayrıştırma hatası: %w", err)
		}

		pathParts := strings.Split(parsedUrl.Path, "/")

		if len(pathParts) < 3 {
			log.Printf("path verisi beklenen formatta değil: %s", parsedUrl.Path)
			continue
		}

		embedID := pathParts[2]
		queryParams := parsedUrl.Query()
		vid := queryParams.Get("vid")

		// Video verilerini al
		apiUrl := fmt.Sprintf("https://%s/api/video/%s?vid=%s", configAnimecix.VideoPlayers[0], embedID, vid)

		response, err := http.Get(apiUrl)
		if err != nil {
			return nil, fmt.Errorf("video verileri alınamadı: %w", err)
		}

		defer response.Body.Close()

		// JSON cevabını çözümle
		respBody, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("video verileri okunamadı: %w", err)
		}

		var videoResp VideoResponse
		err = json.Unmarshal(respBody, &videoResp)
		if err != nil {
			return nil, fmt.Errorf("video verileri ayrıştırılamadı: %w", err)
		}

		// Video URL'lerini listele
		result := make(map[string]interface{})
		streams := make([]interface{}, 0)
		for _, item := range videoResp.URLs {
			entry := map[string]interface{}{
				"label": item.Label,
				"url":   item.URL,
			}

			streams = append(streams, entry)
		}

		result["video_streams"] = streams

		// Altyazı URL'sini ekle
		captions, ok := video["captions"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("'captions' verisi yok veya beklenen formatta değil")
		}

		if len(captions) < 1 {
			result["caption_url"] = nil
			return result, nil
		}

		for _, caption := range captions {
			caption, ok := caption.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("caption verisi beklenen formatta değil")
			}

			lang, ok := caption["language"].(string)
			if ok && lang == "tr" {
				result["caption_url"] = caption["url"]
			} else {
				if len(captions) == 0 {
					return nil, fmt.Errorf("altyazı bulunamadı")
				}
				result["caption_url"] = captions[0].(map[string]interface{})["url"]
			}

			return result, nil
		}
	}

	return nil, fmt.Errorf("video verileri alınamadı")
}
