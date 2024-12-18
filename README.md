<div align="center">
  <h1>Önizleme</h1>
</div>

[preview](https://github.com/user-attachments/assets/fef2191a-50e6-4c21-b2db-9c7043455af8)

**aniTR-CLI:** Hızlı bir şekilde anime araması yapabileceğiniz ve istediğiniz animeyi Türkçe altyazılı izleyebileceğiniz terminal aracıdır 💫 Anime severler için hafif, pratik ve kullanışlı bir çözüm sunar 🚀

![GitHub release (latest by date)](https://img.shields.io/github/v/release/xeyossr/anitr-cli?style=for-the-badge)
![AUR version](https://img.shields.io/aur/version/anitr-cli?style=for-the-badge)

## 💻 Kurulum

Eğer Arch tabanlı bir dağıtım kullanıyorsanız, AUR üzerinden tek bir komut ile indirebilirsiniz:

```bash
yay -S anitr-cli
```

Eğer Arch tabanlı olmayan bir dağıtım kullanıyorsanız projeyi doğrudan kaynağından derleyebilirsiniz.

1. **Repoyu klonlayın:**

```bash
git clone https://github.com/xeyossr/anitr-cli.git
cd anitr-cli
```

2. **Bağımlılıkları yükleyin:**
   AniTR-CLI'yi kullanabilmek için sisteminizde şu araçların yüklü olması gerekir:

- `curl`, `mpv` ve `rofi` (veya Wayland kullanıyorsanız `rofi-wayland`),
- Derleme işlemleri için `gcc`, `cmake` ve `make`.

Bu paketleri dağıtımınıza uygun bir şekilde aşağıdaki komutlarla yükleyebilirsiniz:

```bash
# Arch:
sudo pacman -S curl mpv rofi gcc cmake make

# Debian/Ubuntu:
sudo apt install curl mpv rofi gcc cmake make

# Fedora/RHEL:
sudo dnf install curl mpv rofi gcc cmake make

# OpenSUSE:
sudo zypper install curl mpv rofi gcc cmake make
```

3. **Derleyin:**

Bağımlılıkları kurduktan sonra, AniTR-CLI'yi derlemek için sırasıyla aşağıdaki komutları çalıştırabilirsiniz:

```bash
mkdir -p build && cd build
cmake ..
make
```

ardından, derlenmiş dosyayı `/usr/bin/` dizinine taşımanızı öneririm.

```bash
sudo cp build/anitr-cli /usr/bin/anitr-cli
```

## 👾 Kullanım

Anime izlemeye başlamak için terminalinize `anitr-cli` komutunu yazmanız yeterlidir. Ancak, ilk kez çalıştırdığınızda açılan rofi menüsü şu şekilde gözükecektir:

![preview1](assets/preview-1.png)

"Üstteki videoda menü daha şık görünüyordu, bende neden böyle?"
Bu, rofi için gerekli ayarların yapılmamış olmasından kaynaklanıyor. Eğer rofi'yi daha modern ve şık bir temayla kullanmak istiyorsanız, aşağıdaki adımları takip edin:

1. **Config dosyasını oluşturun:**

```bash
anitr-cli --gen-config
```

2. **Örnek anime teması dosyasını indirin:**
   Örnek olarak kullanılan [anime.rasi](https://raw.githubusercontent.com/xeyossr/anitr-cli/main/rofi%20themes/anime.rasi) temasını indirin ve `~/.config/rofi/anime.rasi` dizinine taşıyın:

```bash
wget -O ~/.config/rofi/anime.rasi https://raw.githubusercontent.com/xeyossr/anitr-cli/main/rofi%20themes/anime.rasi
```

3. **`rofi-flags.conf` dosyasını düzenleyin:**

`~/.config/anitr-cli/rofi-flags.conf` dosyasını açın ve aşağıdaki satırı ekleyin:

```bash
-theme ~/.config/rofi/anime.rasi
```

Bu adımları tamamladıktan sonra, AniTR-CLI artık daha modern ve şık bir görünümle çalışacaktır.
Kendi temalarınızı da oluşturabilir, hatta bu temaları depoya katkı olarak gönderebilirsiniz. Böylece projemize destek olabilirsiniz!

## ⚠️ Karşılaşabileceğiniz sorunlar

Eğer bir sorunla karşılaştıysanız ve aşağıdaki çözümler işe yaramıyorsa, lütfen bir [**issue**](https://github.com/xeyossr/anitr-cli/issue) açarak karşılaştığınız problemi detaylı bir şekilde açıklayın.

### "Bölüm bulunamadı. Tekrar deneyin." hatası

Eğer "Bölüm bulunamadı, tekrar deneyin" gibi bir hata alıyorsanız, muhtemelen bir anime filmi izlemeye çalışıyorsunuzdur. Şu an için AniTR-CLI yalnızca anime dizilerini desteklemektedir. Ancak üzülmeyin, bu özelliği en kısa zamanda eklemeyi planlıyoruz!

Eğer bu özelliği kendiniz ekleyebileceğinize inanıyorsanız, lütfen bir [**pull request**](https://github.com/xeyossr/pulls) göndererek projeye katkıda bulunun. Desteğiniz, AniTR-CLI’yi daha iyi bir hale getirmemiz için çok değerli!

## Katkı

Pull request göndermeden önce lütfen [CONTRIBUTING.md](CONTRIBUTING.md) dosyasını dikkatlice okuduğunuzdan emin olun. Bu dosya, projeye katkıda bulunurken takip etmeniz gereken kuralları ve yönergeleri içermektedir.

## Lisans

Bu proje GNU General Public License v3.0 (GPL-3) altında lisanslanmıştır. Yazılımı bu lisansın koşulları altında kullanmakta, değiştirmekte ve dağıtmakta özgürsünüz. Daha fazla ayrıntı için lütfen [LICENSE](LICENSE) dosyasına bakın.
