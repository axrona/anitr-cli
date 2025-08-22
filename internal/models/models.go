// models paketi, anime verilerini ve ilgili yapılarını tanımlar.
package models

// AnimeSource arayüzü, farklı anime kaynaklarından veri çekme işlevlerini tanımlar.
type AnimeSource interface {
	// Arama sorgusuna göre anime verilerini getirir.
	GetSearchData(query string) ([]Anime, error)
	// Id/Slug ile anime verisini getirir.
	GetAnimeByID(id string) (*Anime, error)
	// Sezon verilerini getirir.
	GetSeasonsData(params SeasonParams) ([]Season, error)
	// Bölüm verilerini getirir.
	GetEpisodesData(params EpisodeParams) ([]Episode, error)
	// İzleme verilerini getirir.
	GetWatchData(params WatchParams) ([]Watch, error)
	// Kaynağın adını döner.
	Source() string
}

// Anime yapısı, bir anime hakkında temel bilgileri içerir.
type Anime struct {
	Title     string                 // Anime başlığı
	ID        *int                   // Anime ID'si (nullable)
	Slug      *string                // URL dostu ad (nullable)
	Type      *string                // Anime türü (dizi/film vb.)
	TitleType *string                // Başlık türü (anime, film vb.)
	ImageURL  string                 // Anime'nin görseli için URL
	Source    string                 // Kaynağın adı
	Extra     map[string]interface{} // Ekstra veri (her türlü bilgi için esnek alan)
}

// Season yapısı, bir anime'nin sezon bilgilerini içerir.
type Season struct {
	Seasons *[]int  // Sezon numaraları (örneğin 1, 2, 3 gibi)
	Count   *int    // Toplam sezon sayısı
	Type    *string // Sezon tipi (normal, movie vb.)
	IsMovie *bool   // Sezonun bir film olup olmadığı
}

// Episode yapısı, bir anime bölümünün bilgilerini içerir.
type Episode struct {
	ID     string                 // Bölüm ID'si
	Title  string                 // Bölüm başlığı
	Number int                    // Bölüm numarası
	Extra  map[string]interface{} // Ekstra veriler
}

// Fansub yapısı, bir anime için Türkçe altyazı ekleyen grup hakkında bilgileri içerir.
type Fansub struct {
	ID         *string // Fansub ID'si (nullable)
	Name       *string // Fansub adı (nullable)
	SecureName *string // Fansub güvenli adı (nullable)
}

// WatchParams yapısı, izleme işlemi için gerekli parametreleri içerir.
type WatchParams struct {
	Slug    *string                 // Anime URL dostu adı (nullable)
	Url     *string                 // İzleme URL'si (nullable)
	Id      *int                    // Anime ID'si (nullable)
	IsMovie *bool                   // Anime'nin film olup olmadığı (nullable)
	Extra   *map[string]interface{} // Ekstra bilgiler (nullable)
}

// FansubParams yapısı, bir bölüm için fansub verilerini filtrelemek amacıyla parametreleri içerir.
type FansubParams struct {
	Slug       *string // Anime URL dostu adı (nullable)
	Id         *int    // Anime ID'si (nullable)
	SeasonNum  *int    // Sezon numarası (nullable)
	EpisodeNum *int    // Bölüm numarası (nullable)
}

// SeasonParams yapısı, bir anime sezonu için gerekli parametreleri içerir.
type SeasonParams struct {
	Slug *string // Anime URL dostu adı (nullable)
	Id   *int    // Anime ID'si (nullable)
}

// EpisodeParams yapısı, bir anime bölümü için gerekli parametreleri içerir.
type EpisodeParams struct {
	Slug     *string // Anime URL dostu adı (nullable)
	SeasonID *int    // Sezon ID'si (nullable)
}

// Watch yapısı, bir anime'yi izlerken gerekli olan izleme bilgilerini içerir.
type Watch struct {
	Labels    []string // Etiketler ("1080", "720p", vb.)
	Urls      []string // İzleme URL'leri
	TRCaption *string  // Türkçe altyazı (nullable)
}
