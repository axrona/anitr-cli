package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xeyossr/anitr-cli/api/animecix"
	"github.com/xeyossr/anitr-cli/internal"
	"github.com/xeyossr/anitr-cli/internal/favorites"
	"github.com/xeyossr/anitr-cli/internal/history"
	"github.com/xeyossr/anitr-cli/internal/player"
	"github.com/xeyossr/anitr-cli/internal/rpc"
	"github.com/xeyossr/anitr-cli/internal/search"
	"github.com/xeyossr/anitr-cli/internal/ui"
	"github.com/xeyossr/anitr-cli/internal/update"
	"github.com/xeyossr/anitr-cli/internal/utils"
)

// Messages
type searchResultMsg struct {
	query   string
	results []map[string]interface{}
	err     string
}

// Bubble Tea Model
type model struct {
	cursor   int
	choices  []string
	selected map[int]struct{}
	state    string
	title    string
	logger   *utils.Logger
	favManager *favorites.FavoritesManager
	histManager *history.HistoryManager
	filterManager *search.FilterManager
	disableRpc *bool
	uiMode   string
	rofiFlags *string
	searchQuery string
	searchResults []map[string]interface{}
	favorites []string
	history []string
	errorMsg string
	loading bool
}

func initialModel(logger *utils.Logger, favManager *favorites.FavoritesManager, histManager *history.HistoryManager, filterManager *search.FilterManager, disableRpc *bool, uiMode string, rofiFlags *string) model {
	return model{
		choices:  []string{"Anime Ara", "Favoriler", "İzleme Geçmişi", "Gelişmiş Arama", "Çıkış"},
		selected: make(map[int]struct{}),
		state:    "main",
		title:    "AniTR-CLI - Ana Menü",
		logger:   logger,
		favManager: favManager,
		histManager: histManager,
		filterManager: filterManager,
		disableRpc: disableRpc,
		uiMode:   uiMode,
		rofiFlags: rofiFlags,
		searchQuery: "",
		errorMsg: "",
		loading: false,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case "main":
			return m.updateMain(msg)
		case "search":
			return m.updateSearch(msg)
		case "favorites":
			return m.updateFavorites(msg)
		case "history":
			return m.updateHistory(msg)
		}
	case searchResultMsg:
		m.loading = false
		if msg.err != "" {
			m.errorMsg = msg.err
			m.searchResults = nil
		} else {
			m.searchResults = msg.results
			m.errorMsg = ""
			// Switch to results view or handle results
			if len(msg.results) > 0 {
				m.errorMsg = fmt.Sprintf("%d sonuç bulundu!", len(msg.results))
			}
		}
	}
	return m, nil
}

func (m model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "enter", " ":
		selected := m.choices[m.cursor]
		switch selected {
		case "Anime Ara":
			m.state = "search"
			m.title = "Anime Ara"
			m.cursor = 0
			m.searchQuery = ""
		case "Favoriler":
			m.state = "favorites"
			m.title = "Favoriler"
			m.cursor = 0
			m = m.loadFavorites()
		case "İzleme Geçmişi":
			m.state = "history"
			m.title = "İzleme Geçmişi"
			m.cursor = 0
			m = m.loadHistory()
		case "Gelişmiş Arama":
			m.errorMsg = "Gelişmiş arama özelliği yakında eklenecek!"
		case "Çıkış":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.state = "main"
		m.title = "AniTR-CLI - Ana Menü"
		m.cursor = 0
		m.choices = []string{"Anime Ara", "Favoriler", "İzleme Geçmişi", "Gelişmiş Arama", "Çıkış"}
		m.errorMsg = ""
	case "enter":
		if m.searchQuery != "" {
			m.loading = true
			return m, m.performSearch()
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		} else if msg.String() == "backspace" && len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	}
	return m, nil
}

func (m model) updateFavorites(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.state = "main"
		m.title = "AniTR-CLI - Ana Menü"
		m.cursor = 0
		m.choices = []string{"Anime Ara", "Favoriler", "İzleme Geçmişi", "Gelişmiş Arama", "Çıkış"}
		m.errorMsg = ""
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.favorites)-1 {
			m.cursor++
		}
	case "enter", " ":
		if len(m.favorites) > 0 && m.cursor < len(m.favorites) {
			m.errorMsg = fmt.Sprintf("%s seçildi! (Bu özellik henüz tam entegre değil)", m.favorites[m.cursor])
		}
	}
	return m, nil
}

func (m model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.state = "main"
		m.title = "AniTR-CLI - Ana Menü"
		m.cursor = 0
		m.choices = []string{"Anime Ara", "Favoriler", "İzleme Geçmişi", "Gelişmiş Arama", "Çıkış"}
		m.errorMsg = ""
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.history)-1 {
			m.cursor++
		}
	case "enter", " ":
		if len(m.history) > 0 && m.cursor < len(m.history) {
			m.errorMsg = fmt.Sprintf("%s seçildi! (Bu özellik henüz tam entegre değil)", m.history[m.cursor])
		}
	}
	return m, nil
}

func (m model) performSearch() tea.Cmd {
	return func() tea.Msg {
		if m.searchQuery == "" {
			return searchResultMsg{query: m.searchQuery, results: nil, err: "Arama terimi boş olamaz"}
		}
		
		// Perform actual search with timeout
		searchData, err := animecix.FetchAnimeSearchData(m.searchQuery)
		if err != nil {
			return searchResultMsg{query: m.searchQuery, results: nil, err: err.Error()}
		}
		
		if len(searchData) == 0 {
			return searchResultMsg{query: m.searchQuery, results: nil, err: "Arama sonucu bulunamadı"}
		}
		
		return searchResultMsg{query: m.searchQuery, results: searchData, err: ""}
	}
}

func (m model) loadFavorites() model {
	favNames, _, err := m.favManager.GetFavoriteNames()
	if err != nil {
		m.errorMsg = fmt.Sprintf("Favoriler yüklenemedi: %v", err)
		m.favorites = []string{}
	} else {
		m.favorites = favNames
		if len(m.favorites) == 0 {
			m.favorites = []string{"Henüz favori anime eklenmemiş"}
		}
	}
	return m
}

func (m model) loadHistory() model {
	histNames, _, err := m.histManager.GetHistoryNames(10)
	if err != nil {
		m.errorMsg = fmt.Sprintf("Geçmiş yüklenemedi: %v", err)
		m.history = []string{}
	} else {
		m.history = histNames
		if len(m.history) == 0 {
			m.history = []string{"Henüz izleme geçmişi yok"}
		}
	}
	return m
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Margin(1, 0)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EE6FF8")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true)

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B")).
		Bold(true)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8BE9FD"))

	s := titleStyle.Render(m.title) + "\n\n"

	switch m.state {
	case "main":
		s += m.renderMainMenu(selectedStyle, normalStyle)
	case "search":
		s += m.renderSearchView(infoStyle, normalStyle)
	case "favorites":
		s += m.renderFavorites(selectedStyle, normalStyle)
	case "history":
		s += m.renderHistory(selectedStyle, normalStyle)
	}

	if m.errorMsg != "" {
		s += "\n" + errorStyle.Render("⚠ " + m.errorMsg)
	}

	if m.loading {
		s += "\n" + successStyle.Render("🔄 Yükleniyor...")
	}

	s += "\n\n" + m.renderHelp()
	return s
}

func (m model) renderMainMenu(selectedStyle, normalStyle lipgloss.Style) string {
	s := ""
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += selectedStyle.Render(cursor + " " + choice)
		} else {
			s += normalStyle.Render(cursor + " " + choice)
		}
		s += "\n"
	}
	return s
}

func (m model) renderSearchView(infoStyle, normalStyle lipgloss.Style) string {
	s := infoStyle.Render("Arama terimi girin:") + "\n\n"
	s += normalStyle.Render("> " + m.searchQuery + "_") + "\n\n"
	
	if m.loading {
		s += "🔄 Yükleniyor..." + "\n"
	}
	
	if m.errorMsg != "" {
		if strings.Contains(m.errorMsg, "sonuç bulundu") {
			s += "✅ " + m.errorMsg + "\n"
		} else {
			s += "❌ " + m.errorMsg + "\n"
		}
	}
	
	if len(m.searchResults) > 0 {
		s += "\n📋 Sonuçlar:\n"
		for i, result := range m.searchResults {
			if i >= 5 { // Limit to first 5 results
				break
			}
			name := internal.GetString(result, "name")
			s += fmt.Sprintf("  %d. %s\n", i+1, name)
		}
		if len(m.searchResults) > 5 {
			s += fmt.Sprintf("  ... ve %d sonuç daha\n", len(m.searchResults)-5)
		}
	}
	
	return s
}

func (m model) renderFavorites(selectedStyle, normalStyle lipgloss.Style) string {
	s := ""
	if len(m.favorites) == 0 {
		s += normalStyle.Render("Henüz favori anime eklenmemiş.")
	} else {
		for i, fav := range m.favorites {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
				s += selectedStyle.Render(cursor + " " + fav)
			} else {
				s += normalStyle.Render(cursor + " " + fav)
			}
			s += "\n"
		}
	}
	return s
}

func (m model) renderHistory(selectedStyle, normalStyle lipgloss.Style) string {
	s := ""
	if len(m.history) == 0 {
		s += normalStyle.Render("Henüz izleme geçmişi yok.")
	} else {
		for i, hist := range m.history {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
				s += selectedStyle.Render(cursor + " " + hist)
			} else {
				s += normalStyle.Render(cursor + " " + hist)
			}
			s += "\n"
		}
	}
	return s
}

func (m model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	switch m.state {
	case "main":
		return helpStyle.Render("↑/↓: Hareket • Enter: Seç • q: Çıkış")
	case "search":
		return helpStyle.Render("Yazın: Arama • Enter: Ara • Esc: Geri • Backspace: Sil")
	case "favorites", "history":
		return helpStyle.Render("↑/↓: Hareket • Enter: Seç • Esc: Geri")
	default:
		return helpStyle.Render("Esc: Geri • q: Çıkış")
	}
}

func FailIfErr(err error, logger *utils.Logger) {
	if err != nil {
		logger.LogError(err)
		log.Fatalf("\033[31mKritik hata: %v\033[0m", err)
	}
}

func checkErr(err error, logger *utils.Logger) bool {
	if err != nil {
		logger.LogError(err)
		fmt.Printf("\n\033[31mHata oluştu: %v\033[0m\nLog detayları: %s\nDevam etmek için bir tuşa basın...\n", err, logger.File.Name())
		fmt.Scanln()
		return false
	}
	return true
}

func isValidImage(url string) bool {
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

func updateWatchApi(episodeData []map[string]interface{}, index, id, seasonIndex, _ int, isMovie bool) (map[string]interface{}, error) {
	var (
		captionData []map[string]string
		captionUrl  string
		err         error
	)

	if isMovie {
		data, movieErr := animecix.AnimeMovieWatchApiUrl(id)
		if movieErr != nil {
			return nil, movieErr
		}

		captionUrlIface := data["caption_url"]
		captionUrl, _ = captionUrlIface.(string)

		streamsIface, ok := data["video_streams"]
		if !ok {
			return nil, fmt.Errorf("video_streams not found")
		}

		rawStreams, _ := streamsIface.([]interface{})
		for _, streamIface := range rawStreams {
			stream, _ := streamIface.(map[string]interface{})
			label := internal.GetString(stream, "label")
			url := internal.GetString(stream, "url")
			captionData = append(captionData, map[string]string{"label": label, "url": url})
		}
	} else {
		indexData := episodeData[index]
		urlData, _ := indexData["url"].(string)
		captionData, err = animecix.AnimeWatchApiUrl(urlData)
		if err != nil {
			return nil, err
		}

		seasonEpisodeIndex := 0
		for i := 0; i < index; i++ {
			if int(episodeData[i]["season_num"].(float64))-1 == seasonIndex {
				seasonEpisodeIndex++
			}
		}
		captionUrl, _ = animecix.FetchTRCaption(seasonIndex, seasonEpisodeIndex, id)

	}

	sort.Slice(captionData, func(i, j int) bool {
		labelI := strings.TrimRight(captionData[i]["label"], "p")
		labelJ := strings.TrimRight(captionData[j]["label"], "p")
		intI, _ := strconv.Atoi(labelI)
		intJ, _ := strconv.Atoi(labelJ)
		return intI > intJ
	})

	labels := []string{}
	urls := []string{}
	for _, item := range captionData {
		labels = append(labels, item["label"])
		urls = append(urls, item["url"])
	}

	return map[string]interface{}{
		"labels":      labels,
		"urls":        urls,
		"caption_url": captionUrl,
	}, nil
}

func main() {
	logger, err := utils.NewLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Initialize managers
	favManager, err := favorites.NewFavoritesManager()
	if err != nil {
		logger.LogError(fmt.Errorf("favori yöneticisi başlatılamadı: %w", err))
	}

	histManager, err := history.NewHistoryManager()
	if err != nil {
		logger.LogError(fmt.Errorf("geçmiş yöneticisi başlatılamadı: %w", err))
	}

	filterManager := search.NewFilterManager()

	log.SetFlags(0)
	uiMode := "tui"

	disableRpc := flag.Bool("disable-rpc", false, "Discord Rich Presence özelliğini devre dışı bırakır.")
	checkUpdate := flag.Bool("update", false, "anitr-cli aracını en son sürüme günceller.")
	printVersion := flag.Bool("version", false, "versiyon")
	rofiMode := flag.Bool("rofi", false, "Rofi arayüzü ile başlatır.")
	rofiFlags := flag.String("rofi-flags", "", "Rofi için flag'ler")
	flag.Parse()

	if *printVersion {
		update.Version()
		return
	}

	if *checkUpdate {
		err := update.RunUpdate()
		FailIfErr(err, logger)
		return
	}

	if *rofiMode {
		uiMode = "rofi"
	}

	update.CheckUpdates()

	// Use Bubble Tea for TUI mode, fallback to old UI for rofi mode
	if uiMode == "tui" {
		m := initialModel(logger, favManager, histManager, filterManager, disableRpc, uiMode, rofiFlags)
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	} else {
		// Fallback to old UI for rofi mode
		for {
			ui.ClearScreen()
			mainMenu := []string{"Anime Ara", "Favoriler", "İzleme Geçmişi", "Gelişmiş Arama", "Çıkış"}
			mainOption, err := ui.SelectionList(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: rofiFlags,
				List:      &mainMenu,
				Label:     "Ana Menü ",
			})
			FailIfErr(err, logger)

			switch mainOption {
			case "Anime Ara":
				handleAnimeSearch(uiMode, rofiFlags, logger, favManager, histManager, filterManager, disableRpc)
			case "Favoriler":
				handleFavorites(uiMode, rofiFlags, logger, favManager, histManager, disableRpc)
			case "İzleme Geçmişi":
				handleHistory(uiMode, rofiFlags, logger, histManager, favManager, disableRpc)
			case "Gelişmiş Arama":
				handleAdvancedSearch(uiMode, rofiFlags, logger, favManager, histManager, filterManager, disableRpc)
			case "Çıkış":
				return
			default:
				return
			}
		}
	}
}

func handleFavorites(uiMode string, rofiFlags *string, logger *utils.Logger, favManager *favorites.FavoritesManager, histManager *history.HistoryManager, disableRpc *bool) {
	ui.ClearScreen()
	favNames, favIDs, err := favManager.GetFavoriteNames()
	if err != nil {
		logger.LogError(err)
		fmt.Printf("\033[31mFavoriler yüklenirken hata: %v\033[0m\n", err)
		return
	}

	if len(favNames) == 0 {
		fmt.Println("\033[33mHenüz favori anime eklenmemiş.\033[0m")
		fmt.Println("Devam etmek için bir tuşa basın...")
		fmt.Scanln()
		return
	}

	selectedFav, err := ui.SelectionList(internal.UiParams{
		Mode:      uiMode,
		RofiFlags: rofiFlags,
		List:      &favNames,
		Label:     "Favori anime seç ",
	})
	if err != nil {
		logger.LogError(err)
		return
	}
	if selectedFav == "" {
		return
	}

	// Find selected anime ID
	selectedIndex := slices.Index(favNames, selectedFav)
	if selectedIndex == -1 {
		return
	}
	selectedAnimeID := favIDs[selectedIndex]

	// Get anime details and start watching
	searchData, err := animecix.FetchAnimeSearchData(fmt.Sprintf("id:%d", selectedAnimeID))
	if err != nil {
		// If direct ID search fails, try to find by name
		re := regexp.MustCompile(`^(.+?) \(ID: (\d+)\)$`)
		match := re.FindStringSubmatch(selectedFav)
		if len(match) >= 2 {
			searchData, err = animecix.FetchAnimeSearchData(match[1])
		}
	}

	if err != nil || searchData == nil {
		fmt.Printf("\033[31mAnime detayları alınamadı: %v\033[0m\n", err)
		fmt.Println("Devam etmek için bir tuşa basın...")
		fmt.Scanln()
		return
	}

	// Find the correct anime from search results
	var selectedAnime map[string]interface{}
	for _, anime := range searchData {
		if int(anime["id"].(float64)) == selectedAnimeID {
			selectedAnime = anime
			break
		}
	}

	if selectedAnime == nil {
		fmt.Println("\033[31mAnime bulunamadı!\033[0m")
		return
	}

	// Check for last watched episode
	lastWatched, err := histManager.GetLastWatchedEpisode(selectedAnimeID)
	if err != nil {
		logger.LogError(err)
	}

	if lastWatched != nil {
		continueMenu := []string{"Kaldığı yerden devam et", "Baştan başla"}
		continueOption, continueErr := ui.SelectionList(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: rofiFlags,
			List:      &continueMenu,
			Label:     fmt.Sprintf("Son izlenen: %s ", lastWatched.EpisodeName),

		})
		if continueErr != nil {
			logger.LogError(continueErr)
			return
		}

		if continueOption == "Kaldığı yerden devam et" {
			// Start from last watched episode
			startWatchingAnime(selectedAnime, lastWatched.EpisodeIndex, uiMode, rofiFlags, logger, favManager, histManager, disableRpc)
			return
		}
	}

	// Start from beginning
	startWatchingAnime(selectedAnime, 0, uiMode, rofiFlags, logger, favManager, histManager, disableRpc)
}

func handleHistory(uiMode string, rofiFlags *string, logger *utils.Logger, histManager *history.HistoryManager, favManager *favorites.FavoritesManager, disableRpc *bool) {
	ui.ClearScreen()
	histNames, histEntries, err := histManager.GetHistoryNames(20)
	if err != nil {
		logger.LogError(err)
		fmt.Printf("\033[31mGeçmiş yüklenirken hata: %v\033[0m\n", err)
		return
	}

	if len(histNames) == 0 {
		fmt.Println("\033[33mHenüz izleme geçmişi yok.\033[0m")
		fmt.Println("Devam etmek için bir tuşa basın...")
		fmt.Scanln()
		return
	}

	// Add clear history option
	histNames = append(histNames, "--- Geçmişi Temizle ---")

	selectedHist, err := ui.SelectionList(internal.UiParams{
		Mode:      uiMode,
		RofiFlags: rofiFlags,
		List:      &histNames,
		Label:     "İzleme geçmişi ",
	})
	if err != nil {
		logger.LogError(err)
		return
	}
	if selectedHist == "" {
		return
	}

	if selectedHist == "--- Geçmişi Temizle ---" {
		confirmMenu := []string{"Evet, temizle", "İptal"}
		confirmResult, confirmErr := ui.SelectionList(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: rofiFlags,
			List:      &confirmMenu,
			Label:     "Geçmişi temizlemek istediğinizden emin misiniz? ",
		})
		if confirmErr != nil {
			logger.LogError(confirmErr)
			return
		}
		if confirmResult == "Evet, temizle" {
			if clearErr := histManager.ClearHistory(); clearErr != nil {
				fmt.Printf("\033[31mGeçmiş temizlenirken hata: %v\033[0m\n", clearErr)
			} else {
				fmt.Println("\033[32mGeçmiş başarıyla temizlendi!\033[0m")
			}
			fmt.Println("Devam etmek için bir tuşa basın...")
			fmt.Scanln()
		}
		return
	}

	// Find selected history entry
	selectedIndex := slices.Index(histNames[:len(histEntries)], selectedHist)
	if selectedIndex == -1 {
		return
	}
	selectedEntry := histEntries[selectedIndex]

	// Get anime details and continue watching
	searchData, err := animecix.FetchAnimeSearchData(selectedEntry.AnimeName)
	if err != nil || searchData == nil {
		fmt.Printf("\033[31mAnime detayları alınamadı: %v\033[0m\n", err)
		fmt.Println("Devam etmek için bir tuşa basın...")
		fmt.Scanln()
		return
	}

	// Find the correct anime from search results
	var selectedAnime map[string]interface{}
	for _, anime := range searchData {
		if int(anime["id"].(float64)) == selectedEntry.AnimeID {
			selectedAnime = anime
			break
		}
	}

	if selectedAnime == nil {
		fmt.Println("\033[31mAnime bulunamadı!\033[0m")
		return
	}

	// Start watching from the episode in history
	startWatchingAnime(selectedAnime, selectedEntry.EpisodeIndex, uiMode, rofiFlags, logger, favManager, histManager, disableRpc)
}

func handleAdvancedSearch(_ string, _ *string, _ *utils.Logger, _ *favorites.FavoritesManager, _ *history.HistoryManager, _ *search.FilterManager, _ *bool) {
	ui.ClearScreen()
	fmt.Println("\033[33mGelişmiş arama özelliği yakında eklenecek!\033[0m")
	fmt.Println("Şu an için normal arama kullanabilirsiniz.")
	fmt.Println("Devam etmek için bir tuşa basın...")
	fmt.Scanln()
}

func startWatchingAnime(selectedAnime map[string]interface{}, startEpisode int, _ string, _ *string, _ *utils.Logger, _ *favorites.FavoritesManager, _ *history.HistoryManager, _ *bool) {
	// This function would contain the anime watching logic
	// For now, we'll just show a message
	animeName := internal.GetString(selectedAnime, "name")
	fmt.Printf("\033[32m%s izlemeye başlanıyor (Bölüm: %d)...\033[0m\n", animeName, startEpisode+1)
	fmt.Println("Bu özellik henüz tam olarak entegre edilmedi.")
	fmt.Println("Devam etmek için bir tuşa basın...")
	fmt.Scanln()
}

func handleAnimeSearch(uiMode string, rofiFlags *string, logger *utils.Logger, favManager *favorites.FavoritesManager, histManager *history.HistoryManager, _ *search.FilterManager, disableRpc *bool) {
	ui.ClearScreen()
	query, err := ui.InputFromUser(internal.UiParams{Mode: uiMode, RofiFlags: rofiFlags, Label: "Anime ara "})
	if err != nil {
		logger.LogError(err)
		return
	}

	searchData, err := animecix.FetchAnimeSearchData(query)
	if err != nil {
		logger.LogError(err)
		return
	}
	if searchData == nil {
		fmt.Println("\033[31m[!] Arama sonucu bulunamadı!\033[0m")
		return
	}

	animeNames := []string{}
	animeTypes := []string{}
	for _, item := range searchData {
		id := int(item["id"].(float64))
		animeNames = append(animeNames, fmt.Sprintf("%s (ID: %d)", item["name"], id))

		ttype := internal.GetString(item, "title_type")
		if strings.ToLower(ttype) == "movie" {
			animeTypes = append(animeTypes, "movie")
		} else {
			animeTypes = append(animeTypes, "tv")
		}
	}

	selectedAnimeName, err := ui.SelectionList(internal.UiParams{Mode: uiMode, RofiFlags: rofiFlags, List: &animeNames, Label: "Anime seç "})
	if err != nil {
		logger.LogError(err)
		return
	}
	if selectedAnimeName == "" {
		return
	}

	selectedIndex := slices.Index(animeNames, selectedAnimeName)
	selectedAnime := searchData[selectedIndex]
	selectedAnimeType := animeTypes[selectedIndex]
	isMovie := selectedAnimeType == "movie"

	posterUrl := internal.GetString(selectedAnime, "poster")
	if !isValidImage(posterUrl) {
		posterUrl = "anitrcli"
	}

	re := regexp.MustCompile(`^(.+?) \(ID: (\d+)\)$`)
	match := re.FindStringSubmatch(selectedAnimeName)
	if len(match) < 3 {
		log.Fatal("ID eşleşmedi")
	}
	selectedAnimeName = match[1]
	selectedAnimeID, _ := strconv.Atoi(match[2])

	var (
		episodes              []map[string]interface{}
		episodeNames          []string
		selectedEpisodeIndex  int
		selectedResolution    string
		selectedResolutionIdx int
		selectedSeasonIndex   int
	)

	if !isMovie {
		episodes, err = animecix.FetchAnimeEpisodesData(selectedAnimeID)
		FailIfErr(err, logger)
		for _, e := range episodes {
			episodeNames = append(episodeNames, internal.GetString(e, "name"))
		}
		selectedSeasonIndex = int(episodes[selectedEpisodeIndex]["season_num"].(float64)) - 1
	} else {
		episodeNames = []string{selectedAnimeName}
		episodes = []map[string]interface{}{
			{
				"name":       selectedAnimeName,
				"season_num": float64(1),
			},
		}
		selectedSeasonIndex = 0
	}

	// Check if anime is in favorites
	isFav, _ := favManager.IsFavorite(selectedAnimeID)
	favText := "Favorilere Ekle"
	if isFav {
		favText = "Favorilerden Çıkar"
	}

	for {
		ui.ClearScreen()
		watchMenu := []string{"İzle", "Çözünürlük seç", favText, "Çık"}
		if !isMovie {
			watchMenu = append([]string{"Sonraki bölüm", "Önceki bölüm", "Bölüm seç"}, watchMenu...)
		}

		option, err := ui.SelectionList(internal.UiParams{
			Mode:      uiMode,
			RofiFlags: rofiFlags,
			List:      &watchMenu,
			Label:     selectedAnimeName,
		})
		if err != nil {
			logger.LogError(err)
			return
		}

		switch option {
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

			// Sezonu her seferinde güncelle
			selectedSeasonIndex = int(episodes[selectedEpisodeIndex]["season_num"].(float64)) - 1

			data, err := updateWatchApi(episodes, selectedEpisodeIndex, selectedAnimeID, selectedSeasonIndex, selectedEpisodeIndex, isMovie)
			if !checkErr(err, logger) {
				continue
			}

			labels := data["labels"].([]string)
			urls := data["urls"].([]string)
			subtitle := data["caption_url"].(string)

			if selectedResolution == "" {
				selectedResolutionIdx = 0
				if len(labels) > 0 {
					selectedResolution = labels[selectedResolutionIdx]
				}
			}

			if selectedResolutionIdx >= len(urls) {
				selectedResolutionIdx = len(urls) - 1
			}

			if !*disableRpc {
				state := selectedAnimeName
				if !isMovie {
					state = fmt.Sprintf("%s (%d/%d)", episodeNames[selectedEpisodeIndex], selectedEpisodeIndex+1, len(episodes))
				}

				if err := rpc.DiscordRPC(internal.RPCParams{
					Details:    selectedAnimeName,
					State:      state,
					LargeImage: posterUrl,
					LargeText:  selectedAnimeName,
				}); err != nil {
					logger.LogError(err)
				}
			}

			playErr := player.Play(urls[selectedResolutionIdx], &subtitle)
			if !checkErr(playErr, logger) {
				continue
			}

			// Add to watch history
			episodeName := selectedAnimeName
			if !isMovie {
				episodeName = episodeNames[selectedEpisodeIndex]
			}
			if err := histManager.AddWatchEntry(selectedAnimeID, selectedAnimeName, selectedEpisodeIndex, episodeName, selectedSeasonIndex, posterUrl); err != nil {
				logger.LogError(fmt.Errorf("geçmiş kaydedilemedi: %w", err))
			}

			// Update favorites last watched if in favorites
			if err := favManager.UpdateLastWatched(selectedAnimeID); err != nil {
				logger.LogError(fmt.Errorf("favori güncelleme hatası: %w", err))
			}

		case "Çözünürlük seç":
			data, err := updateWatchApi(episodes, selectedEpisodeIndex, selectedAnimeID, selectedSeasonIndex, selectedEpisodeIndex, isMovie)
			if !checkErr(err, logger) {
				continue
			}

			labels := data["labels"].([]string)
			selected, err := ui.SelectionList(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: rofiFlags,
				List:      &labels,
				Label:     "Çözünürlük seç ",
			})
			if !checkErr(err, logger) {
				continue
			}

			selectedResolution = selected
			selectedResolutionIdx = slices.Index(labels, selected)

		case "Bölüm seç":
			selected, err := ui.SelectionList(internal.UiParams{
				Mode:      uiMode,
				RofiFlags: rofiFlags,
				List:      &episodeNames,
				Label:     "Bölüm seç ",
			})
			if !checkErr(err, logger) {
				continue
			}

			if selected != "" {
				selectedEpisodeIndex = slices.Index(episodeNames, selected)

				if !isMovie && selectedEpisodeIndex >= 0 && selectedEpisodeIndex < len(episodes) {
					selectedSeasonIndex = int(episodes[selectedEpisodeIndex]["season_num"].(float64)) - 1
				}
			}

			if !*disableRpc {
				totalEpisodes := len(episodes)
				state := fmt.Sprintf("%s (%d/%d)", episodeNames[selectedEpisodeIndex], selectedEpisodeIndex+1, totalEpisodes)
				if err := rpc.DiscordRPC(internal.RPCParams{
					Details:    selectedAnimeName,
					State:      state,
					LargeImage: posterUrl,
					LargeText:  selectedAnimeName,
				}); err != nil {
					logger.LogError(err)
				}
			}

		case "Favorilere Ekle":
			if err := favManager.AddFavorite(selectedAnimeID, selectedAnimeName, posterUrl, selectedAnimeType); err != nil {
				fmt.Printf("\033[31mFavori eklenirken hata: %v\033[0m\n", err)
			} else {
				fmt.Printf("\033[32m%s favorilere eklendi!\033[0m\n", selectedAnimeName)
				isFav = true
				favText = "Favorilerden Çıkar"
			}
			fmt.Println("Devam etmek için bir tuşa basın...")
			fmt.Scanln()

		case "Favorilerden Çıkar":
			if err := favManager.RemoveFavorite(selectedAnimeID); err != nil {
				fmt.Printf("\033[31mFavori çıkarılırken hata: %v\033[0m\n", err)
			} else {
				fmt.Printf("\033[32m%s favorilerden çıkarıldı!\033[0m\n", selectedAnimeName)
				isFav = false
				favText = "Favorilere Ekle"
			}
			fmt.Println("Devam etmek için bir tuşa basın...")
			fmt.Scanln()

		case "Çık":
			return
		default:
			return
		}
	}
}
