<div align="center">
  <h1>Önizleme</h1>
</div>

[preview.mp4](https://github.com/user-attachments/assets/199d940e-14c6-468c-9120-496185ab2217)

<p>
  <img src="assets/discord_rpc_preview.png"/>
</p>

**anitr-cli:** Hızlı bir şekilde anime araması yapabileceğiniz ve istediğiniz animeyi Türkçe altyazılı izleyebileceğiniz terminal aracıdır 💫 Anime severler için hafif, pratik ve kullanışlı bir çözüm sunar 🚀

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/xeyossr/anitr-cli?style=for-the-badge&include_prereleases&label=GitHub%20Release)](https://github.com/xeyossr/anitr-cli/releases)
[![Windows release (latest by date)](https://img.shields.io/github/v/release/mstsecurity/anitr-cli-windows?include_prereleases&display_name=release&label=Windows%20Fork&style=for-the-badge)](https://github.com/mstsecurity/anitr-cli-windows)
[![AUR](https://img.shields.io/aur/version/anitr-cli?style=for-the-badge)](https://aur.archlinux.org/packages/anitr-cli)

## 💻 Kurulum

### 🪟 Windows Kullanıcıları

Bu proje Linux için geliştirilmiştir. **Windows kullanıcıları**, [anitr-cli-windows](https://github.com/mstsecurity/anitr-cli-windows) forkunu kullanabilirler:

> 🔗 [https://github.com/mstsecurity/anitr-cli-windows](https://github.com/mstsecurity/anitr-cli-windows)

Bu fork, Windows uyumluluğu amacıyla oluşturulmuştur ve Windows üzerinde çalışmak için gerekli düzenlemeleri içerir.
Windows sürümünün ortaya çıkmasındaki katkılarından dolayı [@mstsecurity](https://github.com/mstsecurity)'ye teşekkür ederiz.

Forkun geliştirilmesine orijinal proje geliştiricisi [@xeyossr](https://github.com/xeyossr) da katkıda bulunmaktadır.

### 🐧 Linux Kullanıcıları

Eğer Arch tabanlı bir dağıtım kullanıyorsanız, [AUR](https://aur.archlinux.org/packages/anitr-cli) üzerinden tek bir komut ile indirebilirsiniz:

```bash
yay -S anitr-cli
```

Eğer Arch tabanlı olmayan bir dağıtım kullanıyorsanız projeyi [releases](https://github.com/xeyossr/anitr-cli/releases) sayfasından kurabilirsiniz.

```bash
curl -L -o /tmp/anitr-cli https://github.com/xeyossr/anitr-cli/releases/latest/download/anitr-cli
chmod +x /tmp/anitr-cli
sudo mv /tmp/anitr-cli /usr/bin/anitr-cli
```

[Releases](https://github.com/xeyossr/anitr-cli/releases) sayfasından anitr-cli'yi indirdikten sonra, her çalıştırdığınızda yeni bir güncelleme olup olmadığı denetlenecektir. Eğer güncelleme mevcutsa, `anitr-cli --update` komutuyla güncelleyebilirsiniz. Ancak anitr-cli'yi [AUR](https://aur.archlinux.org/packages/anitr-cli) üzerinden kurduysanız, güncelleme için `yay -Sy anitr-cli` komutunu kullanmanız önerilir.

## 👾 Kullanım

```bash
usage: anitr-cli.py [-h] [--source {AnimeciX,OpenAnime}] [--disable-rpc] [--rofi | --tui] [--update]

💫 Terminalden anime izlemek için CLI aracı.

options:
  -h, --help            show this help message and exit
  --source {AnimeciX,OpenAnime}
                        Hangi kaynak ile anime izlemek istediğinizi belirtir. (default: None)
  --disable-rpc         Discord Rich Presence özelliğini devre dışı bırakır. (default: False)
  --rofi                Uygulamanın arayüzünü rofi ile açar. (default: False)
  --tui                 Terminalde TUI arayüzü ile açar. (default: False)
  --update              anitr-cli aracını en son sürüme günceller. (default: False)
```

## Yapılandırma

`anitr-cli`'nin yapılandırma dosyası şurada bulunur: `~/.config/anitr-cli/config`
Aşağıdaki ortam değişkenleri ile uygulamanın davranışını özelleştirebilirsiniz:

```ini
rofi_flags=-i -width 50
rofi_theme=/path/to/theme.rasi
default_ui=rofi
discord_rpc=Enabled
save_position_on_quit=True
```

`ROFI_FLAGS` — Rofi modunda çalıştırırken ek parametreler eklemek için kullanılır.  
`ROFI_THEME` — Rofi arayüzü için özel bir tema belirtmek için kullanılır.  
`DEFAULT_UI` — Uygulamanın varsayılan arayüzünü belirler. `rofi` veya `tui` olarak ayarlanabilir.  
`DISCORD_RPC` - Discord Rich Presence özelliğini aktifleştirir/devre dışı bırakır.  
`SAVE_POSITION_ON_QUIT` - Bir bölümü yarıda bıraksanız bile, MPV kaldığınız saniyeyi hatırlar ve bir sonraki açışınızda tam oradan başlatır.

## Sorunlar

Eğer bir sorunla karşılaştıysanız ve aşağıdaki çözümler işe yaramıyorsa, lütfen bir [**issue**](https://github.com/xeyossr/anitr-cli/issue) açarak karşılaştığınız problemi detaylı bir şekilde açıklayın.

## Katkı

Pull request göndermeden önce lütfen [CONTRIBUTING.md](CONTRIBUTING.md) dosyasını dikkatlice okuduğunuzdan emin olun. Bu dosya, projeye katkıda bulunurken takip etmeniz gereken kuralları ve yönergeleri içermektedir.

## Lisans

Bu proje GNU General Public License v3.0 (GPL-3) altında lisanslanmıştır. Yazılımı bu lisansın koşulları altında kullanmakta, değiştirmekte ve dağıtmakta özgürsünüz. Daha fazla ayrıntı için lütfen [LICENSE](LICENSE) dosyasına bakın.
