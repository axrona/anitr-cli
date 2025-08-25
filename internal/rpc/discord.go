package rpc

import (
	"fmt"

	"github.com/xeyossr/anitr-cli/internal"
	"github.com/xeyossr/go-discordrpc/client"
)

var discordClient *client.Client

// getClient, Tek bir Discord RPC istemci sağlar
func getClient() *client.Client {
	if discordClient == nil {
		discordClient = client.NewClient("1383421771159572600")
	}
	return discordClient
}

// ClientLogin, Discord RPC'ye giriş yapmaya çalışır ve başarı durumunu döner.
func ClientLogin() error {
	c := getClient()
	if err := c.Login(); err != nil {
		return fmt.Errorf("discord rpc login failed: %w", err)
	}
	return nil
}

// DiscordRPC, Discord'a RPC (Remote Procedure Call) aktivitesi güncellemeleri gönderir.
func DiscordRPC(params internal.RPCParams) error {
	c := getClient()

	// Giriş yapıldığından emin ol
	if err := ClientLogin(); err != nil {
		return err
	}

	// Discord aktivitesini ayarla
	err := c.SetActivity(client.Activity{
		Type:       3,                 // Watching
		State:      params.State,      // Aktivite durumu
		Details:    params.Details,    // Aktivite detayları
		LargeImage: params.LargeImage, // Büyük resim
		LargeText:  params.LargeText,  // Büyük resim açıklaması
		SmallImage: params.SmallImage, // Küçük resim
		SmallText:  params.SmallText,  // Küçük resim açıklaması
		Buttons: []*client.Button{ // Butonlar
			{
				Label: "GitHub",
				Url:   "https://github.com/xeyossr/anitr-cli", // GitHub bağlantısı
			},
		},
		Timestamps: &client.Timestamps{
			Start: &params.Timestamp,
		},
	})

	if err == nil {
		return nil
	}

	// Eğer hata oluşursa yeniden bağlan
	_ = c.Logout()
	if err := ClientLogin(); err != nil {
		return fmt.Errorf("discord rpc retry login failed: %w", err)
	}

	// Tekrar dene
	if err := c.SetActivity(client.Activity{
		State:      params.State,
		Details:    params.Details,
		LargeImage: params.LargeImage,
		LargeText:  params.LargeText,
		SmallImage: params.SmallImage,
		SmallText:  params.SmallText,
		Buttons: []*client.Button{
			{
				Label: "GitHub",
				Url:   "https://github.com/xeyossr/anitr-cli",
			},
		},
		Timestamps: &client.Timestamps{
			Start: &params.Timestamp,
		},
	}); err != nil {
		return fmt.Errorf("discord rpc retry set activity failed: %w", err)
	}

	return nil
}

// ClientLogout, Discord RPC'den çıkış yapar
func ClientLogout() error {
	c := getClient()
	if err := c.Logout(); err != nil {
		return fmt.Errorf("discord rpc logout failed: %w", err)
	}
	return nil
}

// RPCDetails, Discord RPC için gerekli parametreleri hazırlar ve döner.
func RPCDetails(details, state, largeimg, largetext string) internal.RPCParams {
	// RPC parametrelerini yapılandır
	return internal.RPCParams{
		Details:    details,
		State:      state,
		LargeImage: largeimg,
		LargeText:  largetext,
	}
}
