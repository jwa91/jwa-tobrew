# Installation

jwa-tobrew publishes tagged GitHub Releases as the release source of truth.
Homebrew and direct downloads consume those release artifacts. `go install`
builds from the same module tag.

## Homebrew

```sh
brew install --cask jwa91/tap/jwa-tobrew
jwa-tobrew version
```

Update:

```sh
brew update
brew upgrade --cask jwa-tobrew
```

Uninstall:

```sh
brew uninstall --cask jwa-tobrew
```

## Direct Download

Download the archive for your platform from the [latest release](https://github.com/jwa91/jwa-tobrew/releases/latest). Asset names use this shape:

- `jwa-tobrew_<version>_darwin_arm64.tar.gz`
- `jwa-tobrew_<version>_darwin_amd64.tar.gz`
- `jwa-tobrew_<version>_linux_arm64.tar.gz`
- `jwa-tobrew_<version>_linux_amd64.tar.gz`

Verify and extract:

```sh
version=0.4.4
asset=jwa-tobrew_${version}_linux_amd64.tar.gz
curl -fsSLO "https://github.com/jwa91/jwa-tobrew/releases/download/v${version}/${asset}"
curl -fsSLO "https://github.com/jwa91/jwa-tobrew/releases/download/v${version}/checksums.txt"
grep "  $asset$" checksums.txt > "$asset.sha256"
shasum -a 256 -c "$asset.sha256"
tar -xzf "$asset"
./jwa-tobrew version
```

On Linux, use `sha256sum -c "$asset.sha256"` if `shasum` is not installed.

## Go

```sh
go install github.com/jwa91/jwa-tobrew/cmd/jwa-tobrew@latest
jwa-tobrew version
```

For a pinned install:

```sh
go install github.com/jwa91/jwa-tobrew/cmd/jwa-tobrew@v0.4.4
```

The Go install path builds from the module tag instead of downloading the
GitHub Release archive, so commit/date metadata may be less complete than
Homebrew or direct-release installs.

## Source Build

For alignment work that must use untagged changes, build from a checkout:

```sh
git clone https://github.com/jwa91/jwa-tobrew.git
cd jwa-tobrew
make build
./bin/jwa-tobrew version
```

Use this path for isolated analysis when you need untagged local changes
instead of the latest released Cask.
