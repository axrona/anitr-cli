#!/usr/bin/env python3

from modules.fetch import animecix, openanime
import modules.player as player
import modules.ui as ui
import modules.utils as utils
import modules.config as config
import modules.discord_rpc as rpc
import argparse
import sys
import re
import requests
import os


def parse_arguments():
    parser = argparse.ArgumentParser(
        description="💫 Terminalden anime izlemek için CLI aracı.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )

    # Kaynak seçimi
    parser.add_argument(
        "--source",
        choices=["AnimeciX", "OpenAnime"],
        type=str,
        help="Hangi kaynak ile anime izlemek istediğinizi belirtir."
    )

    # Discord RPC
    parser.add_argument(
        "--disable-rpc",
        action="store_true",
        help="Discord Rich Presence özelliğini devre dışı bırakır."
    )

    # Arayüz seçenekleri
    mode_group = parser.add_mutually_exclusive_group(required=False)
    mode_group.add_argument(
        "--rofi",
        action="store_true",
        help="Uygulamanın arayüzünü rofi ile açar."
    )
    mode_group.add_argument(
        "--tui",
        action="store_true",
        help="Terminalde TUI arayüzü ile açar."
    )

    # Güncelleme seçeneği
    parser.add_argument(
        "--update",
        action="store_true",
        help="anitr-cli aracını en son sürüme günceller."
    )

    return parser.parse_args()


def is_valid_image_url(url):
    try:
        response = requests.head(url, allow_redirects=True, timeout=3)
        content_type = response.headers.get("Content-Type", "")
        return response.status_code == 200 and "image" in content_type
    except requests.RequestException:
        return False


def save_image_from_url(url, selected_anime_name):
    if not is_valid_image_url(url):
        return None

    # Dosya adı için anime adını küçült ve boşlukları - yap
    safe_name = selected_anime_name.lower().replace(" ", "-")

    # URL'den uzantıyı al (örn: .jpg)
    try:
        _, ext = os.path.splitext(url)

    except Exception as e:
        utils.log_error(config.error_log, e)

    file_path = f"/tmp/{safe_name}{ext}"

    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()

        with open(file_path, "wb") as f:
            f.write(response.content)

        return file_path
    except requests.RequestException:
        return None


def AnimeciX():
    query = ui.search_menu(config.default_ui, "Anime ara >")

    if not query or query == "Çık":
        return

    search_data = animecix().fetch_anime_search_data(query)

    if not search_data:
        ui.show_error(config.default_ui, "Arama sonucu bulunamadı.")
        return

    anime_data = [{"name": item["name"], "id": item["id"], "title_type": item.get(
        "title_type", ""), "type": item.get("type", ""), "poster": item.get("poster", "")} for item in search_data]
    anime_names = [f'{item["name"]} (ID: {item["id"]})' for item in anime_data]
    anime_types = [
        "movie" if (item["title_type"] and item["title_type"].lower() == "movie") or
        (item["type"] and item["type"].lower() == "movie") else "tv"
        for item in anime_data
    ]

    selected_anime_name = ui.select_menu(
        config.default_ui, anime_names, "Anime seç:", True)
    if not selected_anime_name:
        return

    selected_anime_index = anime_names.index(selected_anime_name)
    selected_anime_type = anime_types[selected_anime_index]
    is_movie = selected_anime_type == "movie"

    poster_url = anime_data[selected_anime_index].get("poster")
    poster_url = poster_url if is_valid_image_url(poster_url) else "anitrcli"

    match = re.match(r'(.+) \(ID: (\d+)\)', selected_anime_name)
    if match:
        selected_anime_name = match.group(1)
        selected_anime_id = match.group(2)

    selected_index, selected_label = None, None
    selected_resolution_index = 0
    selected_resolution = None

    def update_watch_api(index, selected_id):
        if is_movie:
            data = animecix().fetch_anime_movie_watch_api_url(selected_id)
            # Movie verisi: {'video_streams': [{'label': '480p', 'url': '...'}, ...]}
            data = data.get("video_streams", [])  # type: ignore

            caption_url = None
            for stream in data:
                if "caption_url" in stream:
                    caption_url = stream["caption_url"]
                    break

        else:
            data = animecix().fetch_anime_watch_api_url(
                anime_episodes_data[index]["url"])
            # Dizi verisi: [{'label': '1080p', 'url': '...'}, ...]

            caption_url = animecix().fetch_tr_caption_url(
                selected_season_index, selected_episode_index, selected_anime_id)

        try:
            data.sort(key=lambda x: int(x['label'][:-1]), reverse=True)
        except Exception as e:
            utils.log_error(config.error_log, e)
            utils.send_notification(
                "anitr-cli", f"anitr-cli bir hatayla karşılaştı. Hata detayları: {config.error_log}", "critical")

        labels = [item['label'] for item in data]
        urls = [item['url'] for item in data]
        return data, labels, urls, caption_url

    if not is_movie:
        anime_episodes_data = animecix().fetch_anime_episodes(selected_anime_id)
        anime_episode_names = [item['name'] for item in anime_episodes_data]
        selected_episode_index = 0
        total_episodes = len(anime_episode_names)
        selected_episode_name = anime_episode_names[selected_episode_index]
        selected_season_index = anime_episodes_data[selected_episode_index]["season_num"] - 1
    else:
        selected_episode_index = 0
        total_episodes = 1
        selected_episode_name = selected_anime_name
        selected_season_index = 0

    anime_series_menu_options = (
        ["Filmi izle", "Çözünürlük seç", "Kaynak değiştir", "Anime ara", "Çık"] if is_movie else
        ["İzle", "Sonraki bölüm", "Önceki bölüm", "Bölüm seç",
            "Çözünürlük seç", "Kaynak değiştir", "Anime ara", "Çık"]
    )

    while True:
        if selected_resolution:
            if not is_movie:
                menu_header = (f"\033[33mOynatılıyor\033[0m: {selected_anime_name} ({selected_resolution}) |"
                               f" {selected_episode_index + 1}/{total_episodes}"
                               if selected_anime_name else "")
            elif is_movie:
                menu_header = (f"\033[33mOynatılıyor\033[0m: {selected_anime_name} ({selected_resolution})"
                               if selected_anime_name else "")

        else:
            if not is_movie:
                menu_header = (f"\033[33mOynatılıyor\033[0m: {selected_anime_name} |"
                               f" {selected_episode_index + 1}/{total_episodes}"
                               if selected_anime_name else "")
            elif is_movie:
                menu_header = (f"\033[33mOynatılıyor\033[0m: {selected_anime_name}"
                               if selected_anime_name else "")

        utils.smart_print(menu_header, "", False, icon=save_image_from_url(
            poster_url, selected_anime_name))

        rpc.log_anime_details(
            details=f"{selected_anime_name}",
            state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                total_episodes})" if not is_movie else f"{selected_anime_name}",
            large_image=poster_url,
            large_text=f"{selected_anime_name}",
            source="AnimeciX",
            source_url="https://anm.cx"
        )

        if config.discord_rpc.lower() == "enabled":
            rpc.start_discord_rpc()

        selected_option = ui.select_menu(
            config.default_ui, anime_series_menu_options, "", False, menu_header)

        if selected_option == "İzle" or selected_option == "Filmi izle":
            watch_api_data, watch_api_labels, watch_api_urls, subtitle_url = update_watch_api(
                selected_episode_index, selected_anime_id)

            if selected_resolution is None:
                if watch_api_labels:
                    selected_index, selected_label = max(
                        enumerate(watch_api_labels), key=lambda x: int(x[1][:-1])
                    )
                    selected_resolution = selected_label
                    selected_resolution_index = selected_index
                else:
                    selected_resolution_index = 0

            while selected_resolution_index >= len(watch_api_urls):
                selected_resolution_index -= 1

            if not is_movie:
                utils.smart_print(
                    f"\033[33mOynatılıyor\033[0m: {selected_episode_name}",
                    f"{selected_anime_name}, {selected_episode_name} ({selected_episode_index+1}/{total_episodes}) oynatılıyor", icon=save_image_from_url(poster_url, selected_anime_name)
                )

            elif is_movie:
                utils.smart_print(
                    f"\033[33mOynatılıyor\033[0m: {selected_anime_name}",
                    f"{selected_anime_name} oynatılıyor", icon=save_image_from_url(poster_url, selected_anime_name)
                )

            selected_season_index = anime_episodes_data[selected_episode_index]["season_num"] - \
                1 if not is_movie else 0
            try:
                watch_url = watch_api_urls[selected_resolution_index]
            except IndexError as e:
                utils.log_error(config.log_error, e)
                utils.show_notification(
                    "anitr-cli", f"anitr-cli bir hatayla karşılaştı. Hata detayları: {config.log_error}", "critical")
                continue

            player.open_with_video_player(
                watch_url, subtitle_url, save_position_on_quit=config.save_position_on_quit
            )
            continue

        elif selected_option == "Sonraki bölüm":
            if selected_episode_index + 1 >= len(anime_episodes_data):
                ui.show_error(config.default_ui, "Zaten son bölümdesiniz.")
                continue
            selected_episode_index += 1
            selected_episode_name = anime_episode_names[selected_episode_index]

            rpc.log_anime_details(
                details=f"{selected_anime_name}",
                state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                    total_episodes})" if not is_movie else f"{selected_anime_name}",
                large_image=poster_url,
                large_text=f"{selected_anime_name}",
                source="AnimeciX",
                source_url="https://anm.cx"
            )

            rpc.update_discord_rpc()

            continue

        elif selected_option == "Önceki bölüm":
            if selected_episode_index == 0:
                ui.show_error(config.default_ui, "Zaten ilk bölümdesiniz.")
                continue
            selected_episode_index -= 1
            selected_episode_name = anime_episode_names[selected_episode_index]

            rpc.log_anime_details(
                details=f"{selected_anime_name}",
                state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                    total_episodes})" if not is_movie else f"{selected_anime_name}",
                large_image=poster_url,
                large_text=f"{selected_anime_name}",
                source="AnimeciX",
                source_url="https://anm.cx"
            )

            rpc.update_discord_rpc()

            continue

        elif selected_option == "Bölüm seç":
            selected_episode_name = ui.select_menu(
                config.default_ui, anime_episode_names, "Bölüm seç:", True)

            if not selected_episode_name:
                continue

            selected_episode_index = anime_episode_names.index(
                selected_episode_name)
            selected_episode_name = anime_episode_names[selected_episode_index]

            rpc.log_anime_details(
                details=f"{selected_anime_name}",
                state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                    total_episodes})" if not is_movie else f"{selected_anime_name}",
                large_image=poster_url,
                large_text=f"{selected_anime_name}",
                source="AnimeciX",
                source_url="https://anm.cx"
            )

            rpc.update_discord_rpc()

            continue

        elif selected_option == "Çözünürlük seç":
            watch_api_data, watch_api_labels, watch_api_urls, subtitle_url = update_watch_api(
                selected_episode_index, selected_anime_id)
            selected_resolution = ui.select_menu(
                config.default_ui, watch_api_labels, "Çözünürlük seç:", False)

            if not selected_resolution:
                continue

            selected_resolution_index = watch_api_labels.index(
                selected_resolution)

        elif selected_option == "Kaynak değiştir":
            selected_source = utils.get_source(ui)
            if selected_source == "AnimeciX (anm.cx)":
                return AnimeciX()
            elif selected_source == "OpenAnime (openani.me)":
                return OpenAnime()
            else:
                return

        elif selected_option == "Anime ara":
            AnimeciX()
            continue

        elif selected_option == "Çık":
            sys.exit()


def OpenAnime():
    query = ui.search_menu(config.default_ui, "Anime ara >")
    if not query or query == "Çık":
        return

    search_data = openanime().search(query)

    if not search_data:
        ui.show_error(config.default_ui, "Arama sonucu bulunamadı.")
        return

    anime_names = []
    for item in search_data:
        if "name" in item and "slug" in item:
            anime_names.append(f'{item["name"]} (ID: {item["slug"]})')

    selected_anime_name = ui.select_menu(
        config.default_ui, anime_names, "Anime seç:", True)

    selected_anime_index = anime_names.index(selected_anime_name)
    poster_url = search_data[selected_anime_index].get('poster')
    poster_url = poster_url if is_valid_image_url(poster_url) else "anitrcli"

    if not selected_anime_name:
        return

    match = re.match(r'(.+) \(ID: (.+)\)', selected_anime_name)
    if match:
        selected_anime_name = match.group(1)
        selected_anime_slug = match.group(2)

    selected_index, selected_label = None, None
    selected_resolution_index = 0
    selected_resolution = None

    seasons_data = openanime().get_seasons(selected_anime_slug)
    anime_type = seasons_data.get("type", "").lower()
    is_movie = anime_type == "movie"

    anime_episodes_data = openanime().get_episodes(selected_anime_slug, is_movie)

    anime_episode_names = []
    for item in anime_episodes_data:
        if is_movie:
            anime_episode_names.append(selected_anime_name)
        else:
            season = item['season']
            episode = item['episode']
            if season == 1:
                anime_episode_names.append(f"{episode}. Bölüm")
            else:
                anime_episode_names.append(
                    f"{season}. Sezon, {episode}. Bölüm")

    if not anime_episodes_data:
        ui.show_error(config.default_ui, "Bu animeye ait bölüm bulunamadı.")
        return

    selected_episode_index = 0
    total_episodes = len(anime_episode_names)
    selected_episode_name = anime_episode_names[selected_episode_index]

    anime_series_menu_options = (
        ["Filmi izle", "Çözünürlük seç", "Kaynak değiştir", "Anime ara", "Çık"] if is_movie else
        ["İzle", "Sonraki bölüm", "Önceki bölüm", "Bölüm seç",
            "Çözünürlük seç", "Kaynak değiştir", "Anime ara", "Çık"]
    )

    def update_watch_api(index):
        episode_data = anime_episodes_data[index]
        data = openanime().get_stream_url(selected_anime_slug,
                                          episode_data["episode"], episode_data["season"])
        if not data:
            return [], [], []

        data.sort(key=lambda x: int(x['resolution']), reverse=True)
        labels = [f"{item['resolution']}p" for item in data]
        urls = [item['url'] for item in data]
        return data, labels, urls

    while True:
        if selected_resolution:
            menu_header = (f"\033[33mOynatılıyor\033[0m: {selected_anime_name} ({selected_resolution}) |"
                           f" {selected_episode_index + 1}/{total_episodes}"
                           if selected_anime_name else "")
        else:
            menu_header = (f"\033[33mOynatılıyor\033[0m: {selected_anime_name} |"
                           f" {selected_episode_index + 1}/{total_episodes}"
                           if selected_anime_name else "")

        utils.smart_print(menu_header, "", False, icon=save_image_from_url(
            poster_url, selected_anime_name))

        rpc.log_anime_details(
            details=f"{selected_anime_name}",
            state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                total_episodes})" if not is_movie else f"{selected_anime_name}",
            large_image=poster_url,
            large_text=f"{selected_anime_name}",
            source="OpenAnime",
            source_url="https://openani.me"
        )

        if config.discord_rpc.lower() == "enabled":
            rpc.start_discord_rpc()

        selected_option = ui.select_menu(
            config.default_ui, anime_series_menu_options, "", False, menu_header)

        if selected_option in ["İzle", "Filmi izle"]:
            watch_api_data, watch_api_labels, watch_api_urls = update_watch_api(
                selected_episode_index)

            if not watch_api_labels:
                ui.show_error(config.default_ui,
                              "Çözünürlük verisi alınamadı.")
                continue

            if selected_resolution is None:
                selected_index, selected_label = max(
                    enumerate(watch_api_labels), key=lambda x: int(x[1][:-1])
                )
                selected_resolution = selected_label
                selected_resolution_index = selected_index

            while selected_resolution_index >= len(watch_api_urls):
                selected_resolution_index -= 1

            utils.smart_print(
                f"\033[33mOynatılıyor\033[0m: {selected_episode_name}",
                f"{selected_anime_name}, {
                    selected_episode_name} ({selected_episode_index+1}/{total_episodes}) oynatılıyor", icon=save_image_from_url(poster_url, selected_anime_name)
            )

            raw_video_url = watch_api_urls[selected_resolution_index]
            season_for_url = 1 if is_movie else anime_episodes_data[selected_episode_index]["season"]
            watch_url = f"{
                openanime().player}/animes/{selected_anime_slug}/{season_for_url}/{raw_video_url}"
            subtitle_url = None
            player.open_with_video_player(
                watch_url, subtitle_url, save_position_on_quit=config.save_position_on_quit
            )
            continue

        elif selected_option == "Sonraki bölüm":
            if selected_episode_index + 1 >= len(anime_episodes_data):
                ui.show_error(config.default_ui, "Zaten son bölümdesiniz.")
                continue
            selected_episode_index += 1
            selected_episode_name = anime_episode_names[selected_episode_index]

            rpc.log_anime_details(
                details=f"{selected_anime_name}",
                state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                    total_episodes})" if not is_movie else f"{selected_anime_name}",
                large_image=poster_url,
                large_text=f"{selected_anime_name}",
                source="OpenAnime",
                source_url="https://openani.me"
            )

            rpc.update_discord_rpc()

            continue

        elif selected_option == "Önceki bölüm":
            if selected_episode_index == 0:
                ui.show_error(config.default_ui, "Zaten ilk bölümdesiniz.")
                continue
            selected_episode_index -= 1
            selected_episode_name = anime_episode_names[selected_episode_index]

            rpc.log_anime_details(
                details=f"{selected_anime_name}",
                state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                    total_episodes})" if not is_movie else f"{selected_anime_name}",
                large_image=poster_url,
                large_text=f"{selected_anime_name}",
                source="OpenAnime",
                source_url="https://openani.me"
            )

            rpc.update_discord_rpc()

            continue

        elif selected_option == "Bölüm seç":
            selected_episode_name = ui.select_menu(
                config.default_ui, anime_episode_names, "Bölüm seç:", True)

            if not selected_episode_name:
                continue

            selected_episode_index = anime_episode_names.index(
                selected_episode_name)

            rpc.log_anime_details(
                details=f"{selected_anime_name}",
                state=f"{selected_episode_name} ({selected_episode_index + 1}/{
                    total_episodes})" if not is_movie else f"{selected_anime_name}",
                large_image=poster_url,
                large_text=f"{selected_anime_name}",
                source="OpenAnime",
                source_url="https://openani.me"
            )

            rpc.update_discord_rpc()

            continue

        elif selected_option == "Çözünürlük seç":
            watch_api_data, watch_api_labels, watch_api_urls = update_watch_api(
                selected_episode_index)
            selected_resolution = ui.select_menu(
                config.default_ui, watch_api_labels, "Çözünürlük seç:", False)

            if not selected_resolution:
                continue

            selected_resolution_index = watch_api_labels.index(
                selected_resolution)

        elif selected_option == "Kaynak değiştir":
            selected_source = utils.get_source(ui)
            if selected_source == "AnimeciX (anm.cx)":
                return AnimeciX()
            elif selected_source == "OpenAnime (openani.me)":
                return OpenAnime()
            else:
                return

        elif selected_option == "Anime ara":
            OpenAnime()
            continue

        elif selected_option == "Çık":
            sys.exit()


def main():
    import modules.update as update
    args = parse_arguments()

    if not args.update:
        if update.check_update_notice():
            utils.send_notification("anitr-cli", update.check_update_notice())

    if args.rofi:
        config.default_ui = "rofi"
    elif args.tui:
        config.default_ui = "tui"
    elif args.update:
        from packaging import version

        latest = update.get_latest_version()
        if version.parse(latest) > version.parse(config.CURRENT_VERSION):
            print(f"Yeni sürüm bulundu: \033[31mv{
                  config.CURRENT_VERSION}\033[0m → \033[32mv{latest}\033[0m")
            update.download_and_replace_binary()
        else:
            print("Zaten en güncel sürümdesiniz.")
        sys.exit()

    if args.disable_rpc:
        config.discord_rpc = "Disabled"

    if args.source:
        if args.source.lower() == "openanime":
            OpenAnime()
        elif args.source.lower() == "animecix":
            AnimeciX()
        else:
            print(f"\033[31m[!] - Geçersiz kaynak\033[0m")
            sys.exit(1)

    selected_source = utils.get_source(ui)
    if selected_source == "AnimeciX (anm.cx)":
        AnimeciX()
    elif selected_source == "OpenAnime (openani.me)":
        OpenAnime()
    else:
        return


if __name__ == "__main__":
    main()
