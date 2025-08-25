// Package update, anitr-cli'nin güncellemeleriyle ilgili bilgileri içerir.
package update

import "fmt"

// GithubRepo, anitr-cli projesinin GitHub üzerindeki kullanıcı/ad şeklindeki yolu.
var GithubRepo string = "axrona/anitr-cli"

// githubAPI, GitHub API üzerinden en son sürüm bilgilerini çeken URL.
var githubAPI string = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GithubRepo)

// repoLink, projenin GitHub sayfasına yönlendiren bağlantı.
var repoLink string = fmt.Sprintf("https://github.com/%s", GithubRepo)

// -v/--version bilgisi için
var version string = "dev"
var buildEnv string = "unknown"
