package rofi

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/axrona/anitr-cli/internal"
)

// isRofiExist, sistemde "rofi" uygulamasının yüklü olup olmadığını kontrol eder
func isRofiExist() error {
	// exec.LookPath ile "rofi" komutunun sistemdeki yolunu kontrol et
	_, err := exec.LookPath("rofi")
	if err != nil {
		// Eğer bulunamazsa hata döndür
		return fmt.Errorf("rofi bulunamadı: %w", err)
	}

	// Eğer "rofi" bulunursa, hata döndürmeden başarılı geri dönüş yapılır
	return nil
}

// SelectionList, verilen seçenekler listesinden bir öğe seçmek için rofi'yi kullanır
func SelectionList(params internal.UiParams) (string, error) {
	// "rofi"nin yüklü olup olmadığını kontrol et
	err := isRofiExist()
	if err != nil {
		return "", errors.New("rofi modunun çalışması için rofi'nin sisteminize yüklü olması gerekmektedir")
	}

	// Rofi komutuna verilecek argümanları hazırla
	args := []string{"-dmenu", "-p", "anitr-cli", "-mesg", params.Label}

	// Eğer rofi özel bayrakları varsa, onları argümanlara ekle
	if params.RofiFlags != nil {
		flags := strings.Split(*params.RofiFlags, " ")
		args = append(args, flags...)
	}

	// Seçenekler listesini "rofi" komutunun standart girişi için uygun formata çevir
	input := bytes.NewBufferString("")
	for _, opt := range *params.List {
		input.WriteString(opt + "\n")
	}

	// "rofi" komutunu çalıştırmak için komut satırını oluştur
	cmd := exec.Command("rofi", args...)
	cmd.Stdin = input // Kullanıcıdan gelen giriş için komutun standart girişini ayarla

	// "rofi" komutunun çıktısını al
	out, err := cmd.Output()
	if err != nil {
		// Eğer komut çalıştırılamazsa hata döndür
		return "", fmt.Errorf("rofi komutu çalıştırılamadı: %w", err)
	}

	// Seçilen öğeyi trimleyip döndür
	selection := strings.TrimSpace(string(out))
	return selection, nil
}

// InputFromUser, kullanıcıdan rofi ile girdi almak için kullanılır
func InputFromUser(params internal.UiParams) (string, error) {
	// "rofi"nin yüklü olup olmadığını kontrol et
	err := isRofiExist()
	if err != nil {
		return "", errors.New("rofi modunun çalışması için rofi'nin sisteminize yüklü olması gerekmektedir")
	}

	// Rofi komutuna verilecek argümanları hazırla
	args := []string{"-dmenu", "-p", "anitr-cli", "-mesg", params.Label}

	// Eğer rofi özel bayrakları varsa, onları argümanlara ekle
	if params.RofiFlags != nil {
		flags := strings.Split(*params.RofiFlags, " ")
		args = append(args, flags...)
	}

	// "rofi" komutunu çalıştırmak için komut satırını oluştur
	cmd := exec.Command("rofi", args...)

	// "rofi" komutunun çıktısını al
	out, err := cmd.Output()
	if err != nil {
		// Eğer komut çalıştırılamazsa hata döndür
		return "", fmt.Errorf("rofi komutu çalıştırılamadı: %w", err)
	}

	// Kullanıcıdan alınan girdiyi trimleyip döndür
	resp := strings.TrimSpace(string(out))
	return resp, nil
}

// Hatayı gösterir
func ShowErrorBox(message string) error {
	fullMessage := "❌ Hata: " + message + "\n"

	// rofi ile göster
	cmd := exec.Command("rofi", "-dmenu", "-p", "Hata Mesajı", "-mesg", fullMessage)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("rofi çalıştırılamadı: %w", err)
	}

	return nil
}
