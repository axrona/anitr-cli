package openanime

import (
	"fmt"
	"strings"

	"github.com/xeyossr/anitr-cli/internal"
	"github.com/xeyossr/anitr-cli/internal/models"
	"github.com/xeyossr/anitr-cli/internal/utils"
)

type OpenAnime struct{}

// OpenAnime API için yapılandırma ayarları
var configOpenAnime = internal.Config{
	BaseUrl:      "https://api.openani.me",                                                                                                                                                                                                                          // API'nin temel URL'si
	VideoPlayers: []string{"https://de2---vn-t9g4tsan-5qcl.yeshi.eu.org"},                                                                                                                                                                                           // Video oynatıcı URL'si
	HttpHeaders:  map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36", "Origin": "https://openani.me", "Referer": "https://openani.me", "Accept": "application/json"}, // HTTP başlıkları
}

// Source, OpenAnime kaynağının adını döner
func (o OpenAnime) Source() string {
	return "OpenAnime"
}

// GetSearchData, verilen sorguya göre anime verilerini döner
func (o OpenAnime) GetSearchData(query string) ([]models.Anime, error) {
	// Türkçe karakterleri ASCII'ye dönüştür ve boşlukları "+" ile değiştir
	normalizedQuery := utils.NormalizeTurkishToASCII(query)
	normalizedQuery = strings.ReplaceAll(normalizedQuery, " ", "+")

	// Arama URL'sini oluştur ve JSON verisini al
	url := fmt.Sprintf("%s/anime/search?q=%s", configOpenAnime.BaseUrl, normalizedQuery)
	data, err := internal.GetJson(url, configOpenAnime.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("arama verileri alınamadı: %w", err)
	}

	var returnData []models.Anime
	// Alınan verileri anime modeline dönüştür
	for _, item := range data.([]interface{}) {
		anime, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("geçersiz anime veri formatı")
		}

		// Anime bilgilerini al
		name, ok := anime["english"].(string)
		if !ok {
			name = ""
		}

		slug, ok := anime["slug"].(string)
		if !ok {
			slug = ""
		}

		poster, ok := anime["pictures"].(map[string]interface{})["avatar"].(string)
		if !ok {
			poster = ""
		}

		// Anime bilgilerini döndür
		returnData = append(returnData, models.Anime{
			Slug:     &slug,
			Title:    name,
			Source:   "openanime",
			ImageURL: poster,
		})
	}

	return returnData, nil
}

// GetAnimeById, doğrudan slug üzerinden anime verilerini döner
func (o OpenAnime) GetAnimeByID(slug string) (*models.Anime, error) {
	url := fmt.Sprintf("%s/anime/%s", configOpenAnime.BaseUrl, slug)
	data, err := internal.GetJson(url, configOpenAnime.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("slug verileri alınamadı: %w", err)
	}

	animeData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("geçersiz anime veri formatı")
	}

	name, ok := animeData["english"].(string)
	if !ok {
		name = ""
	}

	poster, ok := animeData["pictures"].(map[string]interface{})["avatar"].(string)
	if !ok {
		poster = ""
	}

	return &models.Anime{
		Slug:     &slug,
		Title:    name,
		Source:   "openanime",
		ImageURL: poster,
	}, nil
}

// GetSeasonsData, anime için sezon verilerini döner
func (o OpenAnime) GetSeasonsData(params models.SeasonParams) ([]models.Season, error) {
	// Sezon verilerini almak için URL'yi oluştur
	url := fmt.Sprintf("%s/anime/%s", configOpenAnime.BaseUrl, *params.Slug)
	data, err := internal.GetJson(url, configOpenAnime.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("sezon verileri alınamadı: %w", err)
	}

	// Sezon bilgilerini işleyip döndür
	seasonData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("sezon verisi beklenen formatta değil")
	}

	// Sezon sayısını al (Varsa)
	seasonCount, ok := seasonData["numberOfSeasons"].(float64)
	if !ok {
		seasonCount = 1
	}

	contentType := seasonData["type"].(string)
	isMovie := strings.ToLower(contentType) == "movie"

	// Sezon bilgilerini döndür
	return []models.Season{
		{
			Seasons: &[]int{int(seasonCount)},
			Type:    &contentType,
			IsMovie: &isMovie,
		},
	}, nil
}

// GetEpisodesData, sezon için bölüm verilerini döner
func (o OpenAnime) GetEpisodesData(params models.EpisodeParams) ([]models.Episode, error) {
	// Sezon verilerini al
	seasonData, err := o.GetSeasonsData(models.SeasonParams{Slug: params.Slug})
	if err != nil {
		return nil, fmt.Errorf("sezon bilgisi alınamadı: %w", err)
	}

	// Bölüm verilerini al
	var episodes []models.Episode
	seasondata := *seasonData[0].Seasons
	seasonCount := int(seasondata[0])

	// Her bir sezon için bölüm verilerini al
	for season := 1; season <= seasonCount; season++ {
		url := fmt.Sprintf("%s/anime/%s/season/%d", configOpenAnime.BaseUrl, *params.Slug, season)
		data, err := internal.GetJson(url, configOpenAnime.HttpHeaders)
		if err != nil {
			return nil, fmt.Errorf("sezon %d için bölüm verileri alınamadı: %w", season, err)
		}

		seasonInfo, ok := data.(map[string]interface{})["season"].(map[string]interface{})
		if !ok {
			continue
		}

		// Bölüm verilerini işleyip listele
		episodesRaw, ok := seasonInfo["episodes"].([]interface{})
		if !ok {
			continue
		}

		// Her bir bölümü ekle
		for _, episodeRaw := range episodesRaw {
			episode, ok := episodeRaw.(map[string]interface{})
			if !ok {
				continue
			}

			episodeNumber, _ := episode["episodeNumber"].(float64)
			seasonNumber, _ := seasonInfo["season_number"].(float64)
			name := fmt.Sprintf("%d. Sezon, %d. Bölüm", int(seasonNumber), int(episodeNumber))

			episodes = append(episodes, models.Episode{
				Title:  name,
				Number: int(episodeNumber),
				Extra: map[string]interface{}{
					"season_num": seasonNumber,
				},
			})
		}
	}

	return episodes, nil
}

// GetFansubsData, fansub verilerini döner
func (o OpenAnime) GetFansubsData(params models.FansubParams) ([]models.Fansub, error) {
	// Gereksiz boş parametrelerin kontrolü
	if params.Slug == nil || params.SeasonNum == nil || params.EpisodeNum == nil {
		return nil, fmt.Errorf("slug, sezon numarası veya bölüm numarası eksik")
	}

	// Parametreleri al
	slug := *params.Slug
	seasonNum := *params.SeasonNum
	episodeNum := *params.EpisodeNum

	// Fansub verilerini almak için URL'yi oluştur
	url := fmt.Sprintf("%s/anime/%s/season/%d/episode/%d", configOpenAnime.BaseUrl, slug, seasonNum, episodeNum)
	data, err := internal.GetJson(url, configOpenAnime.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("fansub verileri alınamadı: %w", err)
	}

	// Raw fansub verilerini al
	rawFansubs, ok := data.(map[string]interface{})["fansubs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("fansubs verisi eksik veya hatalı")
	}

	// Geçerli fansubları ayıklayıp döndür
	var fansubs []models.Fansub
	for _, f := range rawFansubs {
		fm, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		// 4K çözünürlükleri atla
		is4K, ok := fm["is4K"].(bool)
		if !ok || is4K {
			continue
		}

		id, idOK := fm["id"].(string)
		name, nameOK := fm["name"].(string)
		secureName, secureOK := fm["secureName"].(string)

		// Fansub bilgileri eksikse hata döndür
		if !idOK || !nameOK || !secureOK {
			return nil, fmt.Errorf("fansub bilgisi eksik: %+v", fm)
		}

		// Geçerli fansub'u ekle
		fansubs = append(fansubs, models.Fansub{
			ID:         &id,
			Name:       &name,
			SecureName: &secureName,
		})
	}

	// Eğer hiç geçerli fansub yoksa hata döndür
	if len(fansubs) == 0 {
		return nil, fmt.Errorf("geçerli fansub bulunamadı")
	}

	return fansubs, nil
}

// GetWatchData, izleme verilerini döner
func (o OpenAnime) GetWatchData(req models.WatchParams) ([]models.Watch, error) {
	// Eksik parametre kontrolü
	if req.Slug == nil || req.Extra == nil {
		return nil, fmt.Errorf("slug veya ekstra bilgiler eksik")
	}

	slug := *req.Slug
	extra := *req.Extra

	// Sezon ve bölüm numarasını al
	seasonNum, ok := extra["season_num"].(int)
	if !ok {
		return nil, fmt.Errorf("season_num geçersiz veya eksik")
	}

	episodeNum, ok := extra["episode_num"].(int)
	if !ok {
		return nil, fmt.Errorf("episode_num geçersiz veya eksik")
	}

	// İzleme URL'sini oluştur
	baseURL := fmt.Sprintf("%s/anime/%s/season/%d/episode/%d", configOpenAnime.BaseUrl, slug, int(seasonNum), int(episodeNum))

	// Fansub verilerini al
	fansubs := extra["fansubs"].([]models.Fansub)
	selectedFansubId := extra["selected_fansub_id"].(int)

	// Video URL'sini oluştur
	videoURL := fmt.Sprintf("%s?fansub=%s", baseURL, *fansubs[selectedFansubId].ID)
	data, err := internal.GetJson(videoURL, configOpenAnime.HttpHeaders)
	if err != nil {
		return nil, fmt.Errorf("video bağlantıları alınamadı: %w", err)
	}

	// Bölüm verilerini al
	episodeData, ok := data.(map[string]interface{})["episodeData"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("episodeData eksik veya hatalı")
	}

	// Video dosyalarını al
	files, ok := episodeData["files"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("video dosyaları bulunamadı veya geçersiz")
	}

	var labels []string
	var urls []string

	// Her bir video dosyasını işleyip listele
	for _, f := range files {
		fileData, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		// Dosya URL'sini ve çözünürlüğünü al
		urlRaw, urlOK := fileData["file"].(string)
		url := fmt.Sprintf("%s/animes/%s/%d/%s", configOpenAnime.VideoPlayers[0], slug, seasonNum, urlRaw)
		resolutionVal, resOK := fileData["resolution"].(float64)

		// URL veya çözünürlük eksikse devam et
		if !urlOK || !resOK {
			continue
		}

		// Çözünürlük etiketini ve URL'yi listeye ekle
		labels = append(labels, fmt.Sprintf("%dp", int(resolutionVal)))
		urls = append(urls, url)
	}

	// Eğer geçerli URL yoksa hata döndür
	if len(urls) == 0 {
		return nil, fmt.Errorf("geçerli video bağlantısı bulunamadı")
	}

	// İzleme verilerini döndür
	return []models.Watch{
		{
			Labels: labels,
			Urls:   urls,
		},
	}, nil
}
