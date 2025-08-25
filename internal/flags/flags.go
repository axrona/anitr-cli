package flags

import (
	"runtime"

	"github.com/spf13/cobra"
	"github.com/axrona/anitr-cli/internal/update"
)

type Flags struct {
	DisableRPC   bool
	PrintVersion bool
	RofiMode     bool
	RofiFlags    string
}

func NewFlagsCmd() (*cobra.Command, *Flags) {
	f := &Flags{}

	cmd := &cobra.Command{
		Use:               "anitr-cli",
		Short:             "🚀 Terminalde Türkçe altyazılı anime izleme aracı",
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	cmd.PersistentFlags().BoolVar(&f.DisableRPC, "disable-rpc", false,
		"Discord Rich Presence desteğini devre dışı bırakır.")

	cmd.SetVersionTemplate(update.Version())
	cmd.Version = update.Version()

	if runtime.GOOS == "linux" {
		// Linux'ta rofi ve tui alt komutları eklenir

		// Eski --rofi flag'i (deprecated)
		cmd.PersistentFlags().BoolVarP(&f.RofiMode, "rofi", "r", false,
			"[DEPRECATED] --rofi seçeneği kullanımdan kaldırıldı. Lütfen 'rofi' alt komutunu kullanın.")
		_ = cmd.PersistentFlags().MarkDeprecated("rofi", "Bu bayrak artık kullanılmıyor. Yerine 'rofi' alt komutunu kullanın.")

		// rofi alt komutu
		rofiCmd := &cobra.Command{
			Use:   "rofi",
			Short: "🔹 Rofi arayüzüyle başlatır",
			Long: `Uygulamayı rofi arayüzü ile başlatır.

--rofi-flags bayrağı ile Rofi'ye özel parametreler verilebilir.`,
			Run: func(cmd *cobra.Command, args []string) {
				f.RofiMode = true
			},
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		rofiCmd.Flags().StringVarP(&f.RofiFlags, "rofi-flags", "f", "",
			"Rofi'ye aktarılacak ek parametreler (örnek: --rofi-flags='-theme mytheme')")
		cmd.AddCommand(rofiCmd)

		// tui alt komutu
		tuiCmd := &cobra.Command{
			Use:   "tui",
			Short: "🔹 Terminal (TUI) arayüzüyle başlatır",
			Long:  "Uygulamayı terminal arayüzü (TUI) ile başlatır.",
			Run: func(cmd *cobra.Command, args []string) {
				f.RofiMode = false
			},
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		cmd.AddCommand(tuiCmd)
	} else {
		// Windows'ta rofi yok, otomatik tui modunda başlatılır
		f.RofiMode = false
		// Hiç alt komut ekleme, kullanıcıya seçim sunma
	}

	return cmd, f
}
