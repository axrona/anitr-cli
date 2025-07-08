import requests
from urllib.parse import urlparse, parse_qs


class animecix:
    def __init__(self):
        self.base_url = "https://animecix.tv/"
        self.video_players = ["tau-video.xyz", "sibnet"]
        self.headers = {
            'Accept': 'application/json',
            'User-Agent': 'Mozilla/5.0'
        }

    def _get_json(self, url):
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.RequestException:
            return None

    def fetch_anime_search_data(self, query):
        search_url = f"{self.base_url}secure/search/{query}?type=&limit=20"
        data = self._get_json(search_url)
        if data and 'results' in data:
            return [
                {
                    'name': item.get('name'),
                    'id': item.get('id'),
                    'type': item.get('type'),
                    'title_type': item.get("title_type"),
                    'original_title': item.get("original_title"),
                    'poster': item.get('poster')
                }
                for item in data['results']
            ]
        return []

    def fetch_anime_seasons(self, selected_id):
        url = f"https://mangacix.net/secure/related-videos?episode=1&season=1&titleId={
            selected_id}&videoId=637113"
        json_data = self._get_json(url)
        if json_data and "videos" in json_data:
            videos = json_data["videos"]
            if videos:
                seasons = videos[0].get('title', {}).get('seasons', [])
                if isinstance(seasons, list):
                    return list(range(len(seasons)))
        return []

    def fetch_anime_episodes(self, selected_id):
        seasons = self.fetch_anime_seasons(selected_id)
        episodes = []
        seen_episode_names = set()
        for season_index in seasons:
            url = f"https://mangacix.net/secure/related-videos?episode=1&season={
                season_index + 1}&titleId={selected_id}&videoId=637113"
            data = self._get_json(url)
            if data and 'videos' in data:
                for item in data['videos']:
                    name = item.get('name', "No name field")
                    if name not in seen_episode_names:
                        episode_url = item.get('url', 'No url field')
                        season_num = item.get(
                            'season_num', "No season num field")
                        episodes.append(
                            {'name': name, 'url': episode_url, 'season_num': season_num})
                        seen_episode_names.add(name)
        return episodes

    def fetch_anime_watch_api_url(self, url):
        wtch_url = f"{self.base_url}{url}"
        try:
            response = requests.get(
                wtch_url, headers=self.headers, allow_redirects=True)
            response.raise_for_status()
            final_resp = response.url
            path = urlparse(final_resp).path
            embed_id = path.split('/')[2]
            query = urlparse(final_resp).query
            vid = parse_qs(query).get('vid', [None])[0]
            api_url = f"https://{self.video_players[0]
                                 }/api/video/{embed_id}?vid={vid}"
            wtch_resp = requests.get(api_url)
            wtch_resp.raise_for_status()
            return [{'label': item.get('label', 'No Label field'), 'url': item.get('url', 'No URL field')} for item in wtch_resp.json().get('urls', [])]
        except requests.RequestException:
            return []

    def fetch_tr_caption_url(self, selected_season_index, selected_episode_index, selected_id):
        url = f"https://mangacix.net/secure/related-videos?episode=1&season={
            selected_season_index + 1}&titleId={selected_id}&videoId=637113"
        json_data = self._get_json(url)
        if not json_data or "videos" not in json_data:
            return None
        videos = json_data["videos"]
        if not isinstance(videos, list) or selected_episode_index >= len(videos):
            return None
        video = videos[selected_episode_index]
        captions = video.get("captions")
        if not captions or not isinstance(captions, list):
            return None
        for caption in captions:
            if caption.get("language", "").lower() == "tr":
                return caption.get("url")
        return captions[0].get("url") if captions else None

    def fetch_anime_movie_watch_api_url(self, title_id):
        url = f"{self.base_url}secure/titles/{title_id}?titleId={title_id}"
        movie_headers = self.headers.copy()
        movie_headers["x-e-h"] = "=.a"

        try:
            response = requests.get(url, headers=movie_headers)
            response.raise_for_status()
            data = response.json()

            videos = data.get("title", {}).get("videos", [])

            for video in videos:
                video_url = video.get("url")
                if video_url:
                    resp = requests.get(
                        video_url, headers=self.headers, allow_redirects=True)
                    resp.raise_for_status()
                    final_url = resp.url

                    path = urlparse(final_url).path
                    embed_id = path.split('/')[2]
                    query = urlparse(final_url).query
                    vid = parse_qs(query).get('vid', [None])[0]

                    api_url = f"https://{self.video_players[0]
                                         }/api/video/{embed_id}?vid={vid}"
                    wtch_resp = requests.get(api_url)
                    wtch_resp.raise_for_status()
                    urls = wtch_resp.json().get('urls', [])

                    result = {
                        "video_streams": [{'label': item.get('label', 'No Label field'), 'url': item.get('url', 'No URL field')} for item in urls]
                    }

                    captions = video.get("captions", [])
                    if captions:
                        for caption in captions:
                            if caption.get("language", "").lower() == "tr":
                                result["caption_url"] = caption.get("url")
                                break
                        else:
                            result["caption_url"] = captions[0].get("url")
                    return result
        except requests.RequestException:
            return None


class openanime:
    def __init__(self):
        self.base_url = "https://api.openani.me"
        self.player = "https://de2---vn-t9g4tsan-5qcl.yeshi.eu.org"
        self.headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
            "origin": "https://openani.me",
            "Referer": "https://openani.me",
            "Accept": "application/json",
        }

    def get_json(self, url):
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except requests.RequestException:
            return None

    def search(self, query):
        url = f"{self.base_url}/anime/search?q={query}"
        data = self.get_json(url)
        if data:
            return [{'name': anime.get('english'), 'slug': anime.get('slug'), 'poster': anime.get('pictures', {}).get('avatar')} for anime in data]
        return []

    def get_seasons(self, slug):
        url = f"{self.base_url}/anime/{slug}"
        data = self.get_json(url)
        if data:
            content_type = data.get("type", "").lower()
            is_movie = content_type == "movie"
            return {
                "number_of_seasons": data.get("numberOfSeasons", 0),
                "mal_id": data.get("malID"),
                "type": data.get("type"),
                "is_movie": is_movie
            }
        return {}

    def get_episodes(self, slug, is_movie):
        season_info = self.get_seasons(slug)
        season_count = season_info.get("number_of_seasons", 0)
        all_episodes = []
        if is_movie:
            url = f"{self.base_url}/anime/{slug}/season/1/episode/1"
            data = self.get_json(url)
            if data:
                episode_data = data.get("episodeData", {})
                episode_name = episode_data.get("name", "")
                all_episodes.append({
                    "name": episode_name,
                    "episode": 1,
                    "season": 1
                })
            return all_episodes
        for season in range(1, season_count + 1):
            url = f"{self.base_url}/anime/{slug}/season/{season}"
            data = self.get_json(url)
            if data:
                episodes = data.get("season", {}).get("episodes", [])
                season_number = data["season"]["season_number"]
                for episode in episodes:
                    all_episodes.append({
                        "name": episode.get("name"),
                        "episode": episode.get("episodeNumber"),
                        "season": season_number
                    })
        return sorted(all_episodes, key=lambda ep: (ep["season"], ep["episode"]))

    def get_stream_url(self, slug, episode_number, season_number):
        url = f"{
            self.base_url}/anime/{slug}/season/{season_number}/episode/{episode_number}"
        data = self.get_json(url)
        if data and "episodeData" in data:
            result = []
            for file_data in data["episodeData"]["files"]:
                try:
                    resolution = file_data["resolution"]
                    file_url = file_data["file"]
                    result.append({"resolution": resolution, "url": file_url})
                except KeyError:
                    continue
            return result
        return None
