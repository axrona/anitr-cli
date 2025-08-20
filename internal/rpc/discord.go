package rpc

import (
	"fmt"

	"github.com/hugolgst/rich-go/client"
	"github.com/xeyossr/anitr-cli/internal"
)

// ClientLogin, Discord RPC'ye giriş yapmaya çalışır ve başarı durumunu döner.
func ClientLogin() (bool, error) {
	// Discord RPC'ye giriş yapmayı dene
	err := client.Login("1383421771159572600")
	if err != nil {
		return false, fmt.Errorf("discord rpc login başarısız: %v", err) // Giriş hatası
	}

	return true, nil // Başarılı giriş
}

// DiscordRPC, Discord'a RPC (Remote Procedure Call) aktivitesi güncellemeleri gönderir.
func DiscordRPC(params internal.RPCParams, loggedIn bool) (bool, error) {
	// Eğer Discord'a giriş yapılmamışsa, giriş yap
	if !loggedIn {
		ok, err := ClientLogin()
		if err != nil || !ok {
			return false, fmt.Errorf("discord rpc login başarısız: %v", err)
		}
		loggedIn = true
	}

	// Discord aktivitesini ayarla
	err := client.SetActivity(client.Activity{
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
	})
	// Eğer aktivite güncelleme hatalıysa
	if err != nil {
		loggedIn = false
		ok, err := ClientLogin()
		if err != nil || !ok {
			return false, fmt.Errorf("discord rpc yeniden login başarısız: %v", err)
		}

		err = client.SetActivity(client.Activity{
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
		})
		if err != nil {
			return false, fmt.Errorf("discord rpc retry setactivity başarısız: %v", err)
		}

		loggedIn = true
	}

	return loggedIn, nil // Başarılı RPC güncellemesi
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
