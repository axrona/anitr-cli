package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/axrona/anitr-cli/internal"
	"github.com/axrona/anitr-cli/internal/dl"
	"github.com/axrona/anitr-cli/internal/flags"
	"github.com/axrona/anitr-cli/internal/models"
	"github.com/axrona/anitr-cli/internal/player"
	"github.com/axrona/anitr-cli/internal/rpc"
	"github.com/axrona/anitr-cli/internal/sources/animecix"
	"github.com/axrona/anitr-cli/internal/sources/openanime"
	"github.com/axrona/anitr-cli/internal/ui"
	"github.com/axrona/anitr-cli/internal/ui/tui"
	"github.com/axrona/anitr-cli/internal/update"
	"github.com/axrona/anitr-cli/internal/utils"
	"github.com/spf13/cobra"
)

// updateWatchAPI, seçilen kaynağa (animecix veya openanime) göre bir bölümün izlenebilir URL'lerini ve altyazı bilgilerini getirir.
// Ayrıca varsa TR altyazı URL'sini de döner.
// Params:
// - source: kaynak adı ("animecix", "openanime")
// - episodeData: bölüm listesi
// - index: seçilen bölümün dizindeki yeri
// - id: anime ID'si
// - seasonIndex: sezonun sıfırdan başlayan indeksi
// - selectedFansubIndex: openanime için seçilen fansub'un sırası
// - isMovie: film mi dizi mi
// - slug: openanime için gerekli olan tanımlayıcı
//
// Returns:
// - İzlenebilir kaynakları ve altyazı URL'sini içeren map[string]interface{}
// - Eğer openanime seçildiyse, fansub'ları içeren []models.Fansub
// - Hata (varsa)
func updateWatchAPI(
	source string,
	episodeData []models.Episode,
	index, id, seasonIndex, selectedFansubIndex int,
	isMovie bool,
	slug *string,
) (map[string]interface{}, []models.Fansub, error) {
	var (
		captionData []map[string]string // Video etiketleri ve URL'leri
		fansubData  []models.Fansub     // Fansub listesi (openanime için)
		captionURL  string              // Türkçe altyazı URL'si
		err         error
	)

	switch source {
	case "animecix":
		// Film ise farklı API kullan
		if isMovie {
			data, err := animecix.AnimeMovieWatchApiUrl(id)
			if err != nil {
				return nil, nil, fmt.Errorf("animecix movie API çağrısı başarısız: %w", err)
			}
			// Caption URL ve video stream'leri al
			captionURLIface := data["caption_url"]
			captionURL, _ = captionURLIface.(string)
			streamsIface, ok := data["video_streams"]
			if !ok {
				return nil, nil, fmt.Errorf("video_streams beklenen formatta değil")
			}
			rawStreams, _ := streamsIface.([]interface{})
			for _, streamIface := range rawStreams {
				stream, _ := streamIface.(map[string]interface{})
				label := internal.GetString(stream, "label")
				url := internal.GetString(stream, "url")
				captionData = append(captionData, map[string]string{"label": label, "url": url})
			}
		} else {
			// Dizi bölümü için
			if index < 0 || index >= len(episodeData) {
				return nil, nil, fmt.Errorf("index out of range")
			}
			urlData := episodeData[index].ID
			captionData, err = animecix.AnimeWatchApiUrl(urlData)
			if err != nil {
				return nil, nil, fmt.Errorf("animecix watch API çağrısı başarısız: %w", err)
			}
			// Sezon içerisindeki bölüm indeksini bul
			seasonEpisodeIndex := 0
			for i := 0; i < index; i++ {
				if sn, ok := episodeData[i].Extra["season_num"].(int); ok {
					if sn-1 == seasonIndex {
						seasonEpisodeIndex++
					}
				} else if snf, ok := episodeData[i].Extra["season_num"].(float64); ok {
					if int(snf)-1 == seasonIndex {
						seasonEpisodeIndex++
					}
				}
			}
			// TR altyazı URL'sini almaya çalış
			captionURL, err = animecix.FetchTRCaption(seasonIndex, seasonEpisodeIndex, id)
			if err != nil {
				captionURL = ""
			}
		}

	case "openanime":
		if slug == nil {
			return nil, nil, fmt.Errorf("slug gerekli")
		}
		if index < 0 || index >= len(episodeData) {
			return nil, nil, fmt.Errorf("index out of range")
		}
		ep := episodeData[index]
		seasonNum := 0
		episodeNum := 0

		// Sezon ve bölüm numaralarını al
		if sn, ok := ep.Extra["season_num"].(int); ok {
			seasonNum = sn
		} else if snf, ok := ep.Extra["season_num"].(float64); ok {
			seasonNum = int(snf)
		} else {
			return nil, nil, fmt.Errorf("season_num beklenen formatta değil")
		}
		if en, ok := ep.Extra["episode_num"].(int); ok {
			episodeNum = en
		} else if enf, ok := ep.Extra["episode_num"].(float64); ok {
			episodeNum = int(enf)
		} else {
			episodeNum = ep.Number
		}

		// Fansub listesini al
		fansubParams := models.FansubParams{
			Slug:       slug,
			SeasonNum:  &seasonNum,
			EpisodeNum: &episodeNum,
		}
		fansubData, err = openanime.OpenAnime{}.GetFansubsData(fansubParams)
		if err != nil {
			return nil, nil, fmt.Errorf("fansub data API çağrısı başarısız: %w", err)
		}
		if selectedFansubIndex < 0 || selectedFansubIndex >= len(fansubData) {
			return nil, nil, fmt.Errorf("seçilen fansub indeksi geçersiz")
		}

		// İzlenebilir veri isteği yap
		watchParams := models.WatchParams{
			Slug:    slug,
			Id:      &id,
			IsMovie: &isMovie,
			Extra: &map[string]interface{}{
				"season_num":         seasonNum,
				"episode_num":        episodeNum,
				"fansubs":            fansubData,
				"selected_fansub_id": selectedFansubIndex,
			},
		}
		watches, err := openanime.OpenAnime{}.GetWatchData(watchParams)
		if err != nil {
			return nil, nil, fmt.Errorf("openanime watch data alınamadı: %w", err)
		}
		if len(watches) < 1 {
			return nil, nil, fmt.Errorf("openanime watch data boş")
		}
		w := watches[0]
		captionData = make([]map[string]string, len(w.Labels))
		for i := range w.Labels {
			captionData[i] = map[string]string{
				"label": w.Labels[i],
				"url":   w.Urls[i],
			}
		}
		if w.TRCaption != nil {
			captionURL = *w.TRCaption
		}

	default:
		return nil, nil, fmt.Errorf("geçersiz kaynak: %s", source)
	}

	// Kaliteye göre (etiket sayısal değerine göre) sırala
	sort.Slice(captionData, func(i, j int) bool {
		labelI := strings.TrimRight(captionData[i]["label"], "p")
		labelJ := strings.TrimRight(captionData[j]["label"], "p")
		intI, _ := strconv.Atoi(labelI)
		intJ, _ := strconv.Atoi(labelJ)
		return intI > intJ
	})

	// Etiketleri ve URL'leri ayır
	labels := []string{}
	urls := []string{}
	for _, item := range captionData {
		labels = append(labels, item["label"])
		urls = append(urls, item["url"])
	}

	return map[string]interface{}{
		"labels":      labels,
		"urls":        urls,
		"caption_url": captionURL,
	}, fansubData, nil
}

// getSelectedEpisodesLinks, seçilen bölümlerin sadece seçilmiş çözünürlük URL'lerini döner
func getSelectedEpidodesLinks(
	source string,
	episodes []models.Episode,
	selectedFansubIndex int,
	isMovie bool,
	slug *string,
	selectedResolution string, // kullanıcı seçimi: "720p", "1080p", vb.
	selectedAnimeID int,
) (map[string]string, error) {
	// result[episodeTitle] = url
	result := make(map[string]string)

	for _, ep := range episodes {
		// updateWatchAPI ile tek bölüm için veriyi al
		data, _, err := updateWatchAPI(
			source,
			[]models.Episode{ep}, // tek bölüm
			0,                    // index 0 çünkü slice sadece 1 eleman
			selectedAnimeID,
			0, // sezon index kullanılacaksa güncellenebilir
			selectedFansubIndex,
			isMovie,
			slug,
		)
		if err != nil {
			return nil, fmt.Errorf("[%s] updateWatchAPI hatası: %w", ep.Title, err)
		}

		labelsIface, ok := data["labels"].([]string)
		urlsIface, ok := data["urls"].([]string)
		if !ok {
			return nil, fmt.Errorf("[%s] labels veya urls bulunamadı", ep.Title)
		}
		labels := labelsIface
		urls := urlsIface

		// seçilen çözünürlük için index bul
		resolutionIdx := 0
		for i, label := range labels {
			if label == selectedResolution {
				resolutionIdx = i
				break
			}
		}
		if resolutionIdx >= len(urls) {
			resolutionIdx = len(urls) - 1
		}

		result[ep.Title] = urls[resolutionIdx]
	}

	return result, nil
}

// --- UI ve kullanıcı etkileşimi fonksiyonları ---

// Ana menü
func mainMenu(cfx *App, timestamp time.Time) {
	for {
		// Ekranı temizle
		ui.ClearScreen()

		// Menü seçenekleri
		menuOptions := []string{"Anime Ara", "Kaynak Değiştir", "Geçmiş", "Çık"}

		// Kullanıcıya mevcut kaynağı göster
		label := fmt.Sprintf("Kaynak: %s", *cfx.selectedSource)

		// Seçim al
		selectedChoice, err := showSelection(*cfx, menuOptions, label)
		if err != nil {
			cfx.logger.LogError(err)
			continue
		}

		switch selectedChoice {
		case "Anime Ara":
			// Arama-oynatma döngüsüne gir
			if err := app(cfx, timestamp); err != nil {
				if errors.Is(err, tui.ErrGoBack) {
					continue
				}
				cfx.logger.LogError(err)
			}

		case "Kaynak Değiştir":
			selectedSource, source := selectSource(*cfx.uiMode, *cfx.rofiFlags, *cfx.source, cfx.logger)
			cfx.selectedSource = utils.Ptr(selectedSource)
			cfx.source = utils.Ptr(source)

		case "Geçmiş":
			historySelectedAnime, historyAnimeId, _, err := anitrHistory(internal.UiParams{
				Mode:      *cfx.uiMode,
				RofiFlags: cfx.rofiFlags,
			}, strings.ToLower(*cfx.selectedSource), cfx.historyLimit, cfx.logger)

			if errors.Is(err, tui.ErrGoBack) {
				continue
			}
			if err != nil {
				cfx.logger.LogError(err)
				break
			}

			// Loading spinner başlat
			done := make(chan struct{})
			go ui.ShowLoading(internal.UiParams{
				Mode:      *cfx.uiMode,
				RofiFlags: cfx.rofiFlags,
			}, "Yükleniyor...", done)

			var (
				animeSlug string
				animeId   int
			)

			if strings.ToLower(*cfx.selectedSource) == "openanime" {
				animeSlug = historyAnimeId
			} else {
				animeId, err = strconv.Atoi(historyAnimeId)
			}

			// Bölümleri al
			episodes, episodeNames, isMovie, selectedSeasonIndex, err := getEpisodesAndNames(
				*cfx.source, false, animeId, animeSlug, historySelectedAnime,
			)
			if err != nil {
				close(done) // spinneri durdur

				cfx.logger.LogError(err)

				choice, err := showSelection(App{uiMode: cfx.uiMode, rofiFlags: cfx.rofiFlags}, []string{"Farklı Anime Ara", "Kaynak Değiştir", "Çık"}, fmt.Sprintf("Hata: %s", err.Error()))

				if errors.Is(err, tui.ErrGoBack) {
					continue
				}

				if err != nil {
					os.Exit(0)
				}

				switch choice {
				case "Farklı Anime Ara":
					return
				case "Kaynak Değiştir":
					selectedSource, source := selectSource(*cfx.uiMode, *cfx.rofiFlags, *cfx.source, cfx.logger)
					cfx.selectedSource = utils.Ptr(selectedSource)
					cfx.source = utils.Ptr(source)
					return
				default:
					os.Exit(0)
				}
			}

			// Animenin verilerini çek
			source := *cfx.source
			selectedAnime, err := source.GetAnimeByID(historyAnimeId)
			if err != nil {
				close(done) // spinneri durdur
				cfx.logger.LogError(err)
				return
			}

			// Poster URL al
			posterURL := selectedAnime.ImageURL
			if !utils.IsValidImage(posterURL) {
				posterURL = "anitrcli"
			}

			// Loading spinner durdur
			close(done)

			// Oynatma döngüsü
			newSource, newSelectedSource, err := playAnimeLoop(
				*cfx.source, *cfx.selectedSource, episodes, episodeNames,
				animeId, animeSlug, historySelectedAnime,
				isMovie, selectedSeasonIndex, *cfx.uiMode, *cfx.rofiFlags,
				posterURL, *cfx.disableRPC, timestamp, *cfx.animeHistory, cfx.logger,
			)
			if err != nil {
				cfx.logger.LogError(err)
			}

			// kaynak güncellemesi olursa güncelle
			if newSource != nil && newSelectedSource != "" {
				cfx.source = &newSource
				cfx.selectedSource = &newSelectedSource
			}

		case "Çık":
			os.Exit(0)
		}
	}
}

// Anime geçmişini listeleyen fonksiyon
func anitrHistory(params internal.UiParams, source string, historyLimit int, logger *utils.Logger) (selectedAnime string, animeId string, lastEpisodeIdx int, err error) {
	// Loading spinner başlat
	done := make(chan struct{})
	go ui.ShowLoading(params, "Geçmiş yükleniyor...", done)

	animeHistory, readErr := utils.ReadAnimeHistory()
	if readErr != nil {
		close(done)      // spinner'ı kapat
		ui.ClearScreen() // ekranı temizle
		err = fmt.Errorf("Geçmiş bulunamadı")
		fmt.Printf("\033[31m[!] %s\033[0m\n", err.Error())
		logger.LogError(err)
		time.Sleep(1500 * time.Millisecond)
		return
	}

	sourceData, ok := animeHistory[source]
	if !ok || len(sourceData) == 0 {
		close(done)      // spinner'ı kapat
		ui.ClearScreen() // ekranı temizle
		err = fmt.Errorf("Bu kaynak için geçmiş bulunamadı")
		fmt.Printf("\033[31m[!] %s\033[0m\n", err.Error())
		time.Sleep(1500 * time.Millisecond)
		return
	}

	// slice'e taşı
	type item struct {
		Key       string
		AnimeName string
		AnimeId   string
		Idx       int
		Time      time.Time
	}

	var items []item
	for animeName, entry := range sourceData {
		if entry.LastEpisodeName == "" || entry.LastEpisodeIdx == nil || entry.AnimeId == nil || entry.LastWatched == nil || entry.LastWatched.IsZero() {
			continue
		}
		key := fmt.Sprintf("%s %s", animeName, entry.LastEpisodeName)
		items = append(items, item{
			Key:       key,
			AnimeName: animeName,
			AnimeId:   *entry.AnimeId,
			Idx:       *entry.LastEpisodeIdx,
			Time:      *entry.LastWatched,
		})
	}

	// en yeniden en eskiye sırala
	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.After(items[j].Time)
	})

	// historyLimit ile sınırla
	if historyLimit > len(items) {
		historyLimit = len(items)
	}

	if historyLimit > 0 {
		items = items[:historyLimit]
	}

	close(done) // spinner durdur
	ui.ClearScreen()

	if len(items) == 0 {
		err = fmt.Errorf("Bu kaynak için geçmiş bulunamadı")
		fmt.Printf("\033[31m[!] %s\033[0m\n", err.Error())
		time.Sleep(1500 * time.Millisecond)
		return
	}

	// sadece key stringlerini çıkar
	var keys []string
	for _, it := range items {
		keys = append(keys, it.Key)
	}

	// TUI ile seçim al
	selectedKey, selErr := showSelection(App{
		uiMode:    &params.Mode,
		rofiFlags: params.RofiFlags,
	}, keys, "Geçmiş")
	if selErr != nil {
		err = selErr
		return
	}

	// seçilen animeyi bul
	found := false
	for _, it := range items {
		if it.Key == selectedKey {
			selectedAnime = it.AnimeName
			animeId = it.AnimeId
			lastEpisodeIdx = it.Idx
			found = true
			break
		}
	}
	if !found {
		err = fmt.Errorf("Seçilen anime bulunamadı: %s", selectedKey)
	}

	return
}

// Kullanıcıdan kaynak seçmesini isteyen fonksiyon
func selectSource(uiMode, rofiFlags string, defaultSource models.AnimeSource, logger *utils.Logger) (string, models.AnimeSource) {
	for {
		// Kaynak listesi
		sourceList := []string{"OpenAnime", "AnimeciX"}

		// Kullanıcıdan seçim al
		selectedSource, err := showSelection(
			App{uiMode: &uiMode, rofiFlags: &rofiFlags},
			sourceList,
			"Kaynak seç",
		)

		if errors.Is(err, tui.ErrGoBack) {
			// direkt eski menüye dön
			return defaultSource.Source(), defaultSource
		}

		if err != nil {
			// Kullanıcı iptal ettiyse default'a dön veya menüye at
			logger.LogError(err)
			return "", nil
		}

		// Normalize et
		src := strings.ToLower(strings.TrimSpace(selectedSource))

		// Kaynağı eşleştir
		switch src {
		case "openanime":
			return selectedSource, openanime.OpenAnime{}
		case "animecix":
			return selectedSource, animecix.AnimeCix{}
		default:
			fmt.Printf("\033[31m[!] Geçersiz kaynak seçimi: %s\033[0m\n", selectedSource)
			time.Sleep(1500 * time.Millisecond)
			continue
		}
	}
}

// Kullanıcıdan arama girdisi alır ve API üzerinden sonuçları getirir
func searchAnime(source models.AnimeSource, uiMode string, rofiFlags string, logger *utils.Logger) ([]models.Anime, []string, []string, map[string]models.Anime, error) {
	for {
		// Kullanıcıdan arama kelimesi al
		query, err := ui.InputFromUser(internal.UiParams{Mode: uiMode, RofiFlags: &rofiFlags, Label: "Anime ara "})

		if errors.Is(err, tui.ErrGoBack) {
			// kullanıcı ESC bastı → fonksiyonu çağıran yere geri dön
			return nil, nil, nil, nil, err
		}

		utils.FailIfErr(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: &rofiFlags,
		}, err, logger)

		// Loading spinner başlat
		done := make(chan struct{})
		go ui.ShowLoading(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: &rofiFlags,
		}, "Aranıyor...", done)

		// API üzerinden arama yap
		searchData, err := source.GetSearchData(query)
		if err != nil {
			close(done)      // spinneri durdur
			ui.ClearScreen() // ekranı temizle

			ui.ShowError(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, fmt.Sprintf(
				"%s kaynağına erişilemedi."+"\n\n"+
					"Olası nedenler:\n"+
					"1. VPN açık olabilir\n"+
					"2. Proxy ayarlarından kaynaklı olabilir\n"+
					"3. İnternete bağlı olmayabilirsiniz\n"+
					"4. Bunların hiçbiri değilse API taşınmış olabilir, lütfen GitHub'da issue açarak hatayı bize bildirin.", strings.ToLower(source.Source())))

			os.Exit(1)
		}
		// Hiç sonuç çıkmazsa kullanıcıyı bilgilendir
		if searchData == nil {
			close(done)      // spinneri durdur
			ui.ClearScreen() // ekranı temizle
			fmt.Printf("\033[31m[!] Arama sonucu bulunamadı!\033[0m")
			time.Sleep(1500 * time.Millisecond)
			continue
		}

		// Arama sonuçlarını işleyip ilgili listeleri oluştur
		animeNames := make([]string, 0, len(searchData))
		animeTypes := make([]string, 0, len(searchData))
		animeMap := make(map[string]models.Anime)

		for _, item := range searchData {
			animeNames = append(animeNames, item.Title)
			animeMap[item.Title] = item

			// Anime türünü belirle (tv veya movie)
			if item.TitleType != nil {
				ttype := item.TitleType
				if strings.ToLower(*ttype) == "movie" {
					animeTypes = append(animeTypes, "movie")
				} else {
					animeTypes = append(animeTypes, "tv")
				}
			}
		}

		// Loading spinneri durdur
		close(done)

		return searchData, animeNames, animeTypes, animeMap, nil
	}
}

// Kullanıcının seçtiği animeyi belirler
func selectAnime(animeNames []string, searchData []models.Anime, uiMode string, isMovie bool, rofiFlags string, animeTypes []string, logger *utils.Logger) (models.Anime, bool, int) {
	for {
		ui.ClearScreen()

		// Kullanıcıdan anime seçimi al
		selectedAnimeName, err := showSelection(App{uiMode: &uiMode, rofiFlags: &rofiFlags}, animeNames, "Anime seç ")

		if errors.Is(err, tui.ErrGoBack) {
			// kullanıcı ESC bastı → fonksiyonu çağıran yere geri dön
			return models.Anime{}, false, -1
		}

		utils.FailIfErr(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: &rofiFlags,
		}, err, logger)

		// Geçerli bir anime ismi mi kontrol et
		if !slices.Contains(animeNames, selectedAnimeName) {
			continue
		}

		// Seçilen animeyi bul
		selectedIndex := slices.Index(animeNames, selectedAnimeName)
		selectedAnime := searchData[selectedIndex]

		// Anime türü (movie / tv) güncelleniyor
		if len(animeTypes) > 0 {
			selectedAnimeType := animeTypes[selectedIndex]
			isMovie = selectedAnimeType == "movie"
		}

		return selectedAnime, isMovie, selectedIndex
	}
}

// Seçilen animenin ID veya slug bilgisini döner
func getAnimeIDs(source models.AnimeSource, selectedAnime models.Anime) (int, string) {
	var selectedAnimeID int
	var selectedAnimeSlug string

	// Kaynağa göre ID veya slug alınır
	if strings.ToLower(source.Source()) == "animecix" {
		selectedID := selectedAnime.ID
		selectedAnimeID = *selectedID
	} else if strings.ToLower(source.Source()) == "openanime" {
		selectedSlug := selectedAnime.Slug
		selectedAnimeSlug = *selectedSlug
	}
	return selectedAnimeID, selectedAnimeSlug
}

// Seçilen animeye ait bölümleri getirir, isim listesi oluşturur ve movie olup olmadığını döner
func getEpisodesAndNames(source models.AnimeSource, isMovie bool, selectedAnimeID int, selectedAnimeSlug string, selectedAnimeName string) ([]models.Episode, []string, bool, int, error) {
	var (
		episodes            []models.Episode
		episodeNames        []string
		selectedSeasonIndex int
		err                 error
	)

	// OpenAnime ise sezon verisini alarak movie olup olmadığını kontrol et
	if strings.ToLower(source.Source()) == "openanime" {
		seasonData, err := source.GetSeasonsData(models.SeasonParams{Slug: &selectedAnimeSlug})
		if err != nil {
			return nil, nil, false, 0, fmt.Errorf("sezon verisi alınamadı: %w", err)
		}
		isMovie = *seasonData[0].IsMovie
	}

	if !isMovie {
		// Dizi ise bölüm verilerini al
		episodes, err = source.GetEpisodesData(models.EpisodeParams{SeasonID: &selectedAnimeID, Slug: &selectedAnimeSlug})
		if err != nil {
			return nil, nil, false, 0, fmt.Errorf("bölüm verisi alınamadı: %w", err)
		}

		if len(episodes) == 0 {
			return nil, nil, false, 0, fmt.Errorf("hiçbir bölüm bulunamadı")
		}

		// Bölüm isimlerini listeye ekle
		episodeNames = make([]string, 0, len(episodes))
		for _, e := range episodes {
			episodeNames = append(episodeNames, e.Title)
		}

		// Sezon indeksini belirle
		selectedSeasonIndex = int(episodes[0].Extra["season_num"].(float64)) - 1
	} else {
		// Film ise sadece tek bir bölüm olarak ayarla
		episodeNames = []string{selectedAnimeName}
		episodes = []models.Episode{{
			Title: selectedAnimeName,
			Extra: map[string]interface{}{"season_num": float64(1)},
		}}
		selectedSeasonIndex = 0
	}

	return episodes, episodeNames, isMovie, selectedSeasonIndex, nil
}

// Seçilen animeyi oynatma döngüsünü yönetir.
// Kullanıcıdan izleme seçenekleri alır, çözünürlük/fansub seçtirir, animeyi oynatır ve Discord RPC'yi günceller.
func playAnimeLoop(
	source models.AnimeSource, // Seçilen anime kaynağı (OpenAnime, AnimeciX)
	selectedSource string, // Seçilen kaynak ismi
	episodes []models.Episode, // Tüm bölümler
	episodeNames []string, // Bölüm adları
	selectedAnimeID int, // Seçilen anime ID'si (AnimeciX için)
	selectedAnimeSlug string, // Seçilen anime slug'ı (OpenAnime için)
	selectedAnimeName string, // Seçilen anime ismi
	isMovie bool, // Film mi yoksa dizi mi olduğunu belirtir
	selectedSeasonIndex int, // Seçilen sezonun index'i
	uiMode string, // Arayüz tipi (örneğin terminal, rofi, vs.)
	rofiFlags string, // Rofi için özel bayraklar
	posterURL string, // Poster görseli URL'si (Discord RPC için)
	disableRPC bool, // Discord RPC devre dışı mı?
	timestamp time.Time, // Discord RPC timestamp
	animeHistory utils.AnimeHistory, // Geçmiş veri tipi
	logger *utils.Logger, // Logger
) (models.AnimeSource, string, error) { // Geriye güncel kaynak ve kaynak ismi döner

	selectedEpisodeIndex := 0
	selectedFansubIdx := 0
	selectedResolution := ""
	selectedResolutionIdx := 0

	lastEpisodeIdxP := animeHistory[strings.ToLower(source.Source())][selectedAnimeName].LastEpisodeIdx

	lastEpisodeIdx := -1
	if lastEpisodeIdxP != nil {
		lastEpisodeIdx = *lastEpisodeIdxP
	}
	if lastEpisodeIdx >= 0 && len(episodes) > lastEpisodeIdx+1 {
		// Eğer daha önce izlenmişse bir sonraki bölüm
		selectedEpisodeIndex = lastEpisodeIdx + 1
	}

	for {
		ui.ClearScreen()

		// Kullanıcıya sunulacak menü seçenekleri
		watchMenu := []string{}
		if !isMovie {
			watchMenu = append(watchMenu, "İzle", "Sonraki bölüm", "Önceki bölüm", "Bölüm seç", "Çözünürlük seç", "Bölüm indir")
		} else {
			watchMenu = append(watchMenu, "İzle", "Çözünürlük seç", "Movie indir")
		}

		// OpenAnime için fansub seçimi
		if strings.ToLower(selectedSource) == "openanime" {
			idx := -1
			for i, v := range watchMenu {
				if v == "Bölüm indir" || v == "Movie indir" {
					idx = i
					break
				}
			}

			if idx != -1 {
				watchMenu = append(watchMenu[:idx], append([]string{"Fansub seç"}, watchMenu[idx:]...)...)
			}
		}

		// Genel seçenekler
		watchMenu = append(watchMenu, "Anime ara", "Çık")

		// Seçim arayüzünü göster
		option, err := showSelection(App{uiMode: &uiMode, rofiFlags: &rofiFlags}, watchMenu, selectedAnimeName)

		if errors.Is(err, tui.ErrGoBack) {
			return nil, "", err
		}

		utils.FailIfErr(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: &rofiFlags,
		}, err, logger)

		switch option {

		// Oynatma ve bölüm gezme seçenekleri
		case "İzle", "Sonraki bölüm", "Önceki bölüm":
			ui.ClearScreen()

			if option == "Sonraki bölüm" {
				if selectedEpisodeIndex+1 >= len(episodes) {
					fmt.Println("Zaten son bölümdesiniz.")
					break
				}
				selectedEpisodeIndex++
			} else if option == "Önceki bölüm" {
				if selectedEpisodeIndex <= 0 {
					fmt.Println("Zaten ilk bölümdesiniz.")
					break
				}
				selectedEpisodeIndex--
			}

			// Loading spinner başlat
			done := make(chan struct{})
			go ui.ShowLoading(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, "Başlatılıyor...", done)

			// Güncel sezon bilgisi al
			selectedSeasonIndex = int(episodes[selectedEpisodeIndex].Extra["season_num"].(float64)) - 1

			// API'den oynatma bilgilerini güncelle
			data, _, err := updateWatchAPI(
				strings.ToLower(selectedSource),
				episodes,
				selectedEpisodeIndex,
				selectedAnimeID,
				selectedSeasonIndex,
				selectedFansubIdx,
				isMovie,
				&selectedAnimeSlug,
			)
			if err != nil {
				close(done)      // spinneri durdur
				ui.ClearScreen() // ekranı temizle
				fmt.Printf("\033[31m[!] Bölüm oynatılamadı: %s\033[0m\n", err)
				time.Sleep(1500 * time.Millisecond)
				continue
			}

			labels := data["labels"].([]string)
			urls := data["urls"].([]string)
			subtitle := data["caption_url"].(string)

			// Varsayılan çözünürlük seçimi
			if selectedResolution == "" {
				selectedResolutionIdx = 0
				if len(labels) > 0 {
					selectedResolution = labels[selectedResolutionIdx]
				}
			}
			if selectedResolutionIdx >= len(urls) {
				selectedResolutionIdx = len(urls) - 1
			}

			// MPV başlığı ayarla
			mpvTitle := fmt.Sprintf("%s - %s", selectedAnimeName, episodeNames[selectedEpisodeIndex])
			if isMovie {
				mpvTitle = selectedAnimeName
			}

			// MPV ile oynat
			cmd, socketPath, err := player.Play(player.MPVParams{
				Url:         urls[selectedResolutionIdx],
				SubtitleUrl: &subtitle,
				Title:       mpvTitle,
			})
			if !utils.CheckErr(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, err, logger) {
				close(done) // spinneri durdur
				return source, selectedSource, err
			}

			// MPV’nin çalışıp çalışmadığını kontrol et
			maxAttempts := 10
			mpvRunning := false
			for i := 0; i < maxAttempts; i++ {
				time.Sleep(300 * time.Millisecond)
				if player.IsMPVRunning(socketPath) {
					mpvRunning = true
					break
				}
			}
			if !mpvRunning {
				close(done)      // spinneri durdur
				ui.ClearScreen() // ekranı temizle
				err := fmt.Errorf("MPV başlatılamadı veya zamanında yanıt vermedi")
				logger.LogError(err)
				return source, selectedSource, err
			}

			// Loading spinner durdur
			close(done)

			var stopCh chan struct{}
			if !disableRPC {
				stopCh = make(chan struct{}) // Goroutine'i durdurmak için kanal oluştur
				go updateDiscordRPC(socketPath, episodeNames, selectedEpisodeIndex, selectedAnimeName, selectedSource, posterURL, timestamp, logger, stopCh)
			}

			var selectedAnimeId string

			if strings.ToLower(source.Source()) == "animecix" {
				selectedAnimeId = strconv.Itoa(selectedAnimeID)
			} else {
				selectedAnimeId = selectedAnimeSlug
			}

			// History güncelleme için goroutine
			go utils.UpdateAnimeHistory(socketPath, strings.ToLower(source.Source()), selectedAnimeName, episodeNames[selectedEpisodeIndex], selectedAnimeId, selectedEpisodeIndex, logger)

			// Oynatma işlemi tamamlanana kadar bekle
			err = cmd.Wait()
			if err != nil {
				err = fmt.Errorf("MPV çalışırken hata: %w", err)
				logger.LogError(err)
				return source, selectedSource, err
			}

			if stopCh != nil {
				// MPV kapandı → RPC goroutine'ini durdur
				close(stopCh)
			}

		// Çözünürlük seçme ekranı
		case "Çözünürlük seç":

			// Loading spinner başlat
			done := make(chan struct{})
			go ui.ShowLoading(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, "Hazırlanıyor...", done)

			data, _, err := updateWatchAPI(
				strings.ToLower(selectedSource),
				episodes,
				selectedEpisodeIndex,
				selectedAnimeID,
				selectedSeasonIndex,
				selectedFansubIdx,
				isMovie,
				&selectedAnimeSlug,
			)
			if err != nil {
				close(done)      // spinneri durdur
				ui.ClearScreen() // ekranı temizle

				fmt.Printf("\033[31m[!] Çözünürlükler yüklenemedi.\033[0m\n")
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			labels := data["labels"].([]string)

			// Loading spinner durdur
			close(done)

			selected, err := showSelection(App{uiMode: &uiMode, rofiFlags: &rofiFlags}, labels, "Çözünürlük seç ")

			if errors.Is(err, tui.ErrGoBack) {
				continue
			}

			if !utils.CheckErr(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, err, logger) {
				continue
			}
			selectedResolution = selected
			if !slices.Contains(labels, selected) {
				fmt.Printf("\033[31m[!] Geçersiz çözünürlük seçimi: %s\033[0m\n", selected)
				time.Sleep(1500 * time.Millisecond)
				continue
			}
			selectedResolutionIdx = slices.Index(labels, selected)

		// Bölüm seçimi
		case "Bölüm seç":
			selected, err := showSelection(App{uiMode: &uiMode, rofiFlags: &rofiFlags}, episodeNames, "Bölüm seç ")

			if errors.Is(err, tui.ErrGoBack) {
				continue
			}

			if !utils.CheckErr(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, err, logger) {
				continue
			}
			if slices.Contains(episodeNames, selected) {
				selectedEpisodeIndex = slices.Index(episodeNames, selected)
				if !isMovie && selectedEpisodeIndex >= 0 && selectedEpisodeIndex < len(episodes) {
					selectedSeasonIndex = int(episodes[selectedEpisodeIndex].Extra["season_num"].(float64)) - 1
				}
			} else {
				continue
			}

		// Fansub seçimi (yalnızca OpenAnime için)
		case "Fansub seç":
			// Loading spinner başlat
			done := make(chan struct{})
			go ui.ShowLoading(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, "Hazırlanıyor...", done)

			fansubNames := []string{}

			if strings.ToLower(source.Source()) != "openanime" {
				close(done)      // spinneri durdur
				ui.ClearScreen() // ekranı temizle

				fmt.Println("\033[31m[!] Bu seçenek sadece OpenAnime için geçerlidir.\033[0m")
				time.Sleep(1500 * time.Millisecond)
				continue
			}

			_, fansubData, err := updateWatchAPI(
				strings.ToLower(selectedSource),
				episodes,
				selectedEpisodeIndex,
				selectedAnimeID,
				selectedSeasonIndex,
				selectedFansubIdx,
				isMovie,
				&selectedAnimeSlug,
			)
			if err != nil {
				close(done)      // spinneri durdur
				ui.ClearScreen() // ekranı temizle

				fmt.Printf("\033[31m[!] Fansublar yüklenemedi.\033[0m\n")
				time.Sleep(1000 * time.Millisecond)
				continue
			}

			for _, fansub := range fansubData {
				if fansub.Name != nil {
					fansubNames = append(fansubNames, *fansub.Name)
				}
			}

			// Loading spinner durdur
			close(done)

			selected, err := showSelection(App{uiMode: &uiMode, rofiFlags: &rofiFlags}, fansubNames, "Fansub seç ")

			if errors.Is(err, tui.ErrGoBack) {
				continue
			}

			if !utils.CheckErr(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, err, logger) {
				continue
			}

			if !slices.Contains(fansubNames, selected) {
				fmt.Printf("\033[31m[!] Geçersiz fansub seçimi: %s\033[0m\n", selected)
				time.Sleep(1500 * time.Millisecond)
				continue
			}
			selectedFansubIdx = slices.Index(fansubNames, selected)

		// Movie / Bölüm indir
		case "Bölüm indir", "Movie indir":
			ui.ClearScreen()

			downloader, err := dl.NewDownloader(filepath.Join(utils.VideosDir(), "anitr-cli"))
			if err != nil {
				switch {
				case errors.Is(err, dl.ErrNoDownloader):
					fmt.Printf("\033[31m[!] yt-dlp veya youtube-dl bulunamadı\033[0m\n")
				case errors.Is(err, dl.ErrDirCreate):
					fmt.Printf("\033[31m[!] Klasör oluşturulamadı: %v\033[0m\n", err)
				default:
					fmt.Printf("\033[31m[!] Hata: %v\033[0m\n", err)
				}
				time.Sleep(1500 * time.Millisecond)
				continue
			}

			var choices []string

			if option == "Bölüm indir" {
				choices, err = ui.MultiSelectList(internal.UiParams{
					Mode:      uiMode,
					List:      &episodeNames,
					RofiFlags: &rofiFlags,
					Label:     "Bölüm seç ",
				})

				if errors.Is(err, tui.ErrGoBack) {
					continue
				}

				if err != nil {
					fmt.Printf("\033[31m[!] Seçim listesi oluşturulamadı: %s\033[0m\n", err)
					time.Sleep(1500 * time.Millisecond)
					continue
				}
			} else {
				// Movie ise zaten tek bölüm
				choices = []string{episodeNames[0]}
			}

			// Seçilen bölümleri filtrele
			selectedEpisodes := make([]models.Episode, 0, len(choices))
			episodeNameSet := make(map[string]struct{}, len(choices))

			for _, c := range choices {
				episodeNameSet[c] = struct{}{}
			}

			for _, ep := range episodes {
				if _, ok := episodeNameSet[ep.Title]; ok {
					selectedEpisodes = append(selectedEpisodes, ep)
				}
			}

			// Güncel sezon bilgisi
			if len(selectedEpisodes) > 0 {
				selectedSeasonIndex = int(selectedEpisodes[0].Extra["season_num"].(float64)) - 1
			}

			// Loading spinner başlat
			done := make(chan struct{})
			go ui.ShowLoading(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: &rofiFlags,
			}, "İndiriliyor...", done)

			// Seçilen çözünürlüğe göre tüm bölümlerin URL'lerini al
			links, err := getSelectedEpidodesLinks(
				strings.ToLower(selectedSource),
				selectedEpisodes,
				selectedFansubIdx,
				isMovie,
				&selectedAnimeSlug,
				selectedResolution,
				selectedAnimeID,
			)
			if err != nil {
				close(done)      // spinneri durdur
				ui.ClearScreen() // ekranı temizle

				fmt.Printf("\033[31m[!] Bölüm URL'leri alınamadı: %s\033[0m\n", err)
				time.Sleep(1500 * time.Millisecond)
				continue
			}

			// Loading spinner durdur
			close(done)
			// Yazıyı temizle
			ui.ClearScreen()

			// Downloader ile indirme işlemi
			for _, ep := range selectedEpisodes {
				url, ok := links[ep.Title]
				if !ok {
					fmt.Printf("\033[31m[!] %s için URL bulunamadı.\033[0m\n", ep.Title)
					continue
				}

				episodeNumber, err := utils.ExtractSeasonEpisode(ep.Title)
				if err != nil {
					fmt.Printf("\033[31m[!] %s için bölüm numarası çıkarılamadı: %s\033[0m\n", ep.Title, err)
					continue
				}

				seasonNumber, ok := ep.Extra["season_num"].(float64)
				if !ok {
					logger.LogError(fmt.Errorf("season_num float64 değil"))
				}

				err = downloader.Download(strings.ToLower(source.Source()), selectedAnimeName, url, episodeNumber, int(seasonNumber))
				if err != nil {
					fmt.Printf("\033[31m[!] %s indirilemedi: %s\033[0m\n", ep.Title, err)
				}
			}

		// Yeni bir anime aramak için menü
		case "Anime ara":
			for {
				choice, err := showSelection(App{uiMode: &uiMode, rofiFlags: &rofiFlags}, []string{"Bu kaynakla devam et", "Kaynak değiştir", "Çık"}, fmt.Sprintf("Arama kaynağı: %s", selectedSource))

				if errors.Is(err, tui.ErrGoBack) {
					break
				}

				if err != nil {
					logger.LogError(fmt.Errorf("seçim listesi oluşturulamadı: %w", err))
					continue
				}

				switch choice {
				case "Bu kaynakla devam et":
					// Hiçbir işlem yapma
				case "Kaynak değiştir":
					selectedSource, source = selectSource(uiMode, rofiFlags, source, logger)
				case "Çık":
					os.Exit(0)
				default:
					fmt.Printf("\033[31m[!] Geçersiz seçim: %s\033[0m\n", choice)
					time.Sleep(1500 * time.Millisecond)
					continue
				}

				return source, selectedSource, nil
			}

		// Çıkış seçeneği
		case "Çık":
			os.Exit(0)

		default:
			return source, selectedSource, nil
		}
	}
}

// Discord RPC'yi güncelleyerek anime oynatma durumunu Discord'a yansıtır
func updateDiscordRPC(socketPath string, episodeNames []string, selectedEpisodeIndex int,
	selectedAnimeName, selectedSource, posterURL string, timestamp time.Time, logger *utils.Logger, stopCh <-chan struct{},
) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			// Stop sinyali geldi → Discord RPC'yi kapat
			rpc.ClientLogout()
			return
		case <-ticker.C:
			// Eğer MPV çalışmıyorsa da RPC'yi kapat ve çık
			if !player.IsMPVRunning(socketPath) {
				rpc.ClientLogout()
				return
			}

			// MPV duraklatma durumu
			isPaused, _ := player.GetMPVPausedStatus(socketPath)

			// MPV süre ve konum
			durationVal, _ := player.MPVSendCommand(socketPath, []interface{}{"get_property", "duration"})
			timePosVal, _ := player.MPVSendCommand(socketPath, []interface{}{"get_property", "time-pos"})
			duration, ok1 := durationVal.(float64)
			timePos, ok2 := timePosVal.(float64)
			if !ok1 || !ok2 {
				fmt.Println("süre veya zaman konumu parse edilemedi")
				continue
			}

			formatTime := func(seconds float64) string {
				total := int(seconds + 0.5)
				hours := total / 3600
				minutes := (total % 3600) / 60
				secs := total % 60
				if hours > 0 {
					return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
				}
				return fmt.Sprintf("%02d:%02d", minutes, secs)
			}

			state := fmt.Sprintf("%s (%s / %s)", episodeNames[selectedEpisodeIndex], formatTime(timePos), formatTime(duration))
			if isPaused {
				state += " (Paused)"
			}

			params := internal.RPCParams{
				Type:       3,
				Details:    selectedAnimeName,
				State:      state,
				SmallImage: strings.ToLower(selectedSource),
				SmallText:  selectedSource,
				LargeImage: posterURL,
				LargeText:  selectedAnimeName,
				Timestamp:  timestamp,
			}

			if err := rpc.DiscordRPC(params); err != nil {
				logger.LogError(fmt.Errorf("DiscordRPC hatası: %w", err))
				continue
			}
		}
	}
}

// Uygulama durumu ve ayarlarını saklayan struct
type App struct {
	source         *models.AnimeSource
	selectedSource *string
	uiMode         *string
	rofiFlags      *string
	disableRPC     *bool
	animeHistory   *utils.AnimeHistory
	historyLimit   int
	logger         *utils.Logger
}

// Kullanıcıdan bir seçim almak için kullanılan fonksiyon
func showSelection(cfx App, list []string, label string) (string, error) {
	return ui.SelectionList(internal.UiParams{
		Mode:      *cfx.uiMode,
		RofiFlags: cfx.rofiFlags,
		List:      &list,
		Label:     label,
	})
}

// Uygulamanın ana fonksiyonu, anime seçimi, oynatma ve hata yönetimini içerir
func app(cfx *App, timestamp time.Time) error {
	for {
		// Anime arama işlemi yapılır
		searchData, animeNames, animeTypes, _, err := searchAnime(*cfx.source, *cfx.uiMode, *cfx.rofiFlags, cfx.logger)

		if errors.Is(err, tui.ErrGoBack) {
			return err
		}

		isMovie := false

		// Kullanıcıdan anime seçimi yapılması istenir
		selectedAnime, isMovie, animeidx := selectAnime(animeNames, searchData, *cfx.uiMode, isMovie, *cfx.rofiFlags, animeTypes, cfx.logger)

		if animeidx == -1 {
			continue
		}

		// Loading spinner başlat
		done := make(chan struct{})
		go ui.ShowLoading(internal.UiParams{
			Mode:      *cfx.uiMode,
			RofiFlags: cfx.rofiFlags,
		}, "Yükleniyor...", done)

		// Poster URL'si alınır ve geçersizse varsayılan bir URL kullanılır
		posterURL := selectedAnime.ImageURL
		if !utils.IsValidImage(posterURL) {
			posterURL = "anitrcli"
		}

		// Seçilen animeye ait ID ve slug alınır
		selectedAnimeID, selectedAnimeSlug := getAnimeIDs(*cfx.source, selectedAnime)

		// Anime bölümleri alınır
		episodes, episodeNames, isMovie, selectedSeasonIndex, err := getEpisodesAndNames(
			*cfx.source, isMovie, selectedAnimeID, selectedAnimeSlug, selectedAnime.Title,
		)
		// Hata durumunda kullanıcıya seçenek sunulur
		if err != nil {
			// Loading spinner durdur
			close(done)
			// Hatayı logla
			cfx.logger.LogError(err)

			choice, err := showSelection(App{uiMode: cfx.uiMode, rofiFlags: cfx.rofiFlags}, []string{"Farklı Anime Ara", "Kaynak Değiştir", "Çık"}, fmt.Sprintf("Hata: %s", err.Error()))
			if err != nil {
				os.Exit(0)
			}

			// Kullanıcının seçimine göre işlem yapılır
			switch choice {
			case "Farklı Anime Ara":
				return nil // Üst döngüye geri dön
			case "Kaynak Değiştir":
				selectedSource, source := selectSource(*cfx.uiMode, *cfx.rofiFlags, *cfx.source, cfx.logger)
				cfx.selectedSource = utils.Ptr(selectedSource)
				cfx.source = utils.Ptr(source)
				return nil
			default:
				os.Exit(0)
			}
		}

		// Loading spinneri durdur
		close(done)

		// Oynatma döngüsüne girilir
		newSource, newSelectedSource, err := playAnimeLoop(
			*cfx.source, *cfx.selectedSource, episodes, episodeNames,
			selectedAnimeID, selectedAnimeSlug, selectedAnime.Title,
			isMovie, selectedSeasonIndex, *cfx.uiMode, *cfx.rofiFlags,
			posterURL, *cfx.disableRPC, timestamp, *cfx.animeHistory, cfx.logger,
		)

		if errors.Is(err, tui.ErrGoBack) {
			continue
		}

		// Kaynak değiştiyse güncellenir
		if newSource != *cfx.source || newSelectedSource != *cfx.selectedSource {
			cfx.source = &newSource
			cfx.selectedSource = &newSelectedSource
			return nil
		}
	}
}

// Ana uygulama döngüsünü yöneten fonksiyon
func runMain(cmd *cobra.Command, f *flags.Flags, uiMode string, logger *utils.Logger) {
	// RPC'yi devre dışı bırakma bayrağı ayarlanır
	disableRPC := f.DisableRPC

	// Güncellemeleri kontrol et
	update.CheckUpdates()

	// Geçmişi yükle
	animeHistory, err := utils.ReadAnimeHistory()
	if err != nil {
		logger.LogError(fmt.Errorf(fmt.Sprintf("Geçmiş yüklenemedi: %s", err)))
	}

	// Uygulama durumunu başlat
	currentApp := &App{
		source:         utils.Ptr(models.AnimeSource(openanime.OpenAnime{})),
		selectedSource: utils.Ptr("OpenAnime"),
		uiMode:         &uiMode,
		rofiFlags:      &f.RofiFlags,
		disableRPC:     &disableRPC,
		animeHistory:   &animeHistory,
		historyLimit:   0,
		logger:         logger,
	}

	// Configi yükle
	cfg, err := utils.LoadConfig(filepath.Join(utils.ConfigDir(), "config.json"))
	if err == nil {
		if cfg.DefaultSource != "" {
			// Config'te default_source varsa, onu kullan
			switch strings.ToLower(cfg.DefaultSource) {
			case "openanime":
				currentApp.source = utils.Ptr(models.AnimeSource(openanime.OpenAnime{}))
				currentApp.selectedSource = utils.Ptr("OpenAnime")
			case "animecix":
				currentApp.source = utils.Ptr(models.AnimeSource(animecix.AnimeCix{}))
				currentApp.selectedSource = utils.Ptr("AnimeciX")
			}
		} else {
			// Config'te default_source yoksa OpenAnime kullan
			currentApp.source = utils.Ptr(models.AnimeSource(openanime.OpenAnime{}))
			currentApp.selectedSource = utils.Ptr("OpenAnime")
		}

		// Config'de disable_rpc ayarı varsa
		if cfg.DisableRPC != nil {
			currentApp.disableRPC = cfg.DisableRPC
		}

		// history_limit ayarı (default: 0 yani unlimited)
		currentApp.historyLimit = cfg.HistoryLimit
	}

	if cmd.Flags().Changed("disable-rpc") {
		currentApp.disableRPC = &disableRPC
	}

	timestamp := time.Now()

	for {
		mainMenu(currentApp, timestamp)
	}
}

// Uygulama komutlarını çalıştıran giriş fonksiyonu
func runApp() {
	logger, err := utils.NewLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Close()
	log.SetFlags(0)

	rootCmd, f := flags.NewFlagsCmd()

	commands := rootCmd.Commands()

	if runtime.GOOS != "linux" {
		// Windows ve Mac'te alt komut yok, doğrudan tui modunda çalıştır
		rootCmd.Run = func(cmd *cobra.Command, args []string) {
			f.RofiMode = false
			runMain(rootCmd, f, "tui", logger)
		}
	} else {
		// Linux için alt komutlar varsa ayarla
		var rofiCmd, tuiCmd *cobra.Command
		if len(commands) > 0 {
			rofiCmd = commands[0]
		}
		if len(commands) > 1 {
			tuiCmd = commands[1]
		}

		if rofiCmd != nil {
			rofiCmd.Run = func(cmd *cobra.Command, args []string) {
				f.RofiMode = true
				runMain(rootCmd, f, "rofi", logger)
			}
		}

		if tuiCmd != nil {
			tuiCmd.Run = func(cmd *cobra.Command, args []string) {
				f.RofiMode = false
				runMain(rootCmd, f, "tui", logger)
			}
		}

		rootCmd.Run = func(cmd *cobra.Command, args []string) {
			f.RofiMode = false
			runMain(rootCmd, f, "tui", logger)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	// Uygulamayı başlat
	runApp()
}
