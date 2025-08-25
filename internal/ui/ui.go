package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/axrona/anitr-cli/internal"
	"github.com/axrona/anitr-cli/internal/ui/rofi"
	"github.com/axrona/anitr-cli/internal/ui/tui"
)

// Ekranı temizler
func ClearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// Kullanıcıya seçim listesi gösterir
// Mode rofi ise rofi arayüzü, değilse tui kullanılır
func SelectionList(params internal.UiParams) (string, error) {
	if params.Mode == "rofi" {
		response, err := rofi.SelectionList(params)
		if err != nil {
			return "", fmt.Errorf("rofi seçim listesi oluşturulamadı: %w", err)
		}
		return response, nil
	}

	response, err := tui.SelectionList(params)
	if err != nil {
		if !errors.Is(err, tui.ErrQuit) {
			return "", fmt.Errorf("tui seçim listesi oluşturulamadı: %w", err)
		} else {
			os.Exit(1)
		}
	}
	return response, nil
}

// Kullanıcıdan input almak için
// rofi ya da tui üzerinden alınır
func InputFromUser(params internal.UiParams) (string, error) {
	if params.Mode == "rofi" {
		response, err := rofi.InputFromUser(params)
		if err != nil {
			return "", fmt.Errorf("rofi kullanıcı girişi alınamadı: %w", err)
		}
		return response, nil
	}

	response, err := tui.InputFromUser(params)
	if err != nil {
		if !errors.Is(err, tui.ErrQuit) {
			return "", fmt.Errorf("tui kullanıcı girişi alınamadı: %w", err)
		} else {
			os.Exit(1)
		}
	}
	return response, nil
}

// Kullanıcıya checkbox gösterir. Rofi ile nasıl yapacağımı bilmediğimden olduğu gibi bıraktım
func MultiSelectList(params internal.UiParams) ([]string, error) {
	if params.Mode == "rofi" {
		response, err := rofi.SelectionList(params)
		if err != nil {
			return []string{}, fmt.Errorf("rofi seçim listesi oluşturulamadı: %w", err)
		}
		return []string{response}, nil
	}

	response, err := tui.MultiSelectList(params)
	if err != nil {
		if !errors.Is(err, tui.ErrQuit) {
			return []string{}, fmt.Errorf("tui checkbox listesi oluşturulamadı: %w", err)
		} else {
			os.Exit(1)
		}
	}
	return response, nil
}

// Hata gösterir
func ShowError(params internal.UiParams, message string) {
	if params.Mode == "rofi" {
		err := rofi.ShowErrorBox(message)
		if err != nil {
			fmt.Printf("❌ Hata: %s\n", message)
		}
	} else {
		tui.ShowErrorBox(message)
	}
}

// Spinner (rofiye yok tabi)
func ShowLoading(params internal.UiParams, message string, done chan struct{}) {
	if params.Mode == "tui" {
		tui.ShowSpinner(message, done)
	}
}
