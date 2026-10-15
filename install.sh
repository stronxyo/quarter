#!/bin/sh
# Installs Quarter on macOS or Linux:
#
#   curl -fsSL https://raw.githubusercontent.com/stronxyo/quarter/main/install.sh | sh
#
# It downloads the latest release for this system from GitHub, checks it
# against the release's checksums, and puts the quarter program in
# ~/.local/bin. If that folder isn't on your PATH yet, it adds it in your
# shell's settings file (~/.zshrc, ~/.bashrc or ~/.bash_profile, fish's
# conf.d, or ~/.profile), so typing quarter works in a new terminal.
# Optional settings, as environment variables:
#   QUARTER_VERSION         a version to install instead of the latest (v0.1.0)
#   QUARTER_INSTALL_DIR     where to put the program (default ~/.local/bin)
#   QUARTER_NO_MODIFY_PATH  set to 1 to leave your shell's settings alone
#   QUARTER_OS, QUARTER_ARCH  the build to fetch (darwin or linux; amd64 or arm64)
#   QUARTER_BASE_URL        where release files come from (for testing)
set -eu

repo="stronxyo/quarter"
dir="${QUARTER_INSTALL_DIR:-$HOME/.local/bin}"

say() { printf '%s\n' "$*"; }
fail() { printf 'quarter install: %s\n' "$*" >&2; exit 1; }

# fetch URL FILE downloads URL into FILE with curl or wget.
fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$1" -O "$2"
  else
    fail "needs curl or wget"
  fi
}

os="${QUARTER_OS:-}"
if [ -z "$os" ]; then
  case "$(uname -s)" in
    Darwin) os=darwin ;;
    Linux) os=linux ;;
    *) fail "this script is for macOS and Linux; on Windows use: irm https://raw.githubusercontent.com/$repo/main/install.ps1 | iex" ;;
  esac
fi
arch="${QUARTER_ARCH:-}"
if [ -z "$arch" ]; then
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) fail "no build for this processor ($(uname -m)); see https://github.com/$repo/releases" ;;
  esac
fi

version="${QUARTER_VERSION:-}"
if [ -z "$version" ]; then
  # The latest release's page redirects to its tag: no API call, no limit.
  if command -v curl >/dev/null 2>&1; then
    url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
  else
    url=$(wget -q --max-redirect=5 --server-response --spider "https://github.com/$repo/releases/latest" 2>&1 | sed -n 's/^ *[Ll]ocation: *//p' | tail -n 1 | tr -d '\r')
  fi
  version="${url##*/}"
  case "$version" in
    v[0-9]*) ;;
    *) fail "couldn't find the latest release (got '$version')" ;;
  esac
fi
base="${QUARTER_BASE_URL:-https://github.com/$repo/releases/download/$version}"
archive="quarter_${version#v}_${os}_${arch}.tar.gz"

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t quarter)
trap 'rm -rf "$tmp"' EXIT

say "Downloading Quarter $version for $os/$arch..."
fetch "$base/$archive" "$tmp/$archive"
fetch "$base/checksums.txt" "$tmp/checksums.txt"

want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || fail "$archive isn't in the release's checksums"
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$tmp/$archive" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  got=$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')
else
  fail "needs sha256sum or shasum to check the download"
fi
[ "$got" = "$want" ] || fail "the download doesn't match its checksum; nothing was installed"

tar -xzf "$tmp/$archive" -C "$tmp" quarter
mkdir -p "$dir"
mv -f "$tmp/quarter" "$dir/quarter"
chmod 755 "$dir/quarter"
say "Installed $dir/quarter"

# Typing quarter has to just work. Already on the PATH: done.
case ":$PATH:" in
  *":$dir:"*)
    first=$(command -v quarter 2>/dev/null || true)
    if [ -n "$first" ] && [ "$first" != "$dir/quarter" ]; then
      say "Note: typing quarter starts $first, which comes first on your PATH. Remove that one to use this install."
    fi
    say "Type quarter to start."
    exit 0
    ;;
esac

# by_hand says what to do when the settings file can't be changed here.
by_hand() {
  say ""
  say "To start Quarter by typing quarter, add this line to your shell's settings"
  say "(~/.zshrc on macOS, ~/.bashrc on most Linux), then open a new terminal:"
  say "  export PATH=\"$dir:\$PATH\""
  say "Or start it now with: $dir/quarter"
  exit 0
}

[ -z "${QUARTER_NO_MODIFY_PATH:-}" ] || by_hand
# A quote, $, ` or \ in the folder's name can't go into a settings line safely.
case "$dir" in
  *[\'\"\$\`\\]*) by_hand ;;
esac

# The file a new terminal reads, for the shell new terminals open ($SHELL),
# as rustup and uv do it.
line="export PATH=\"$dir:\$PATH\""
next="Open a new terminal and type quarter to start."
case "${SHELL:-}" in
  */zsh) rc="${ZDOTDIR:-$HOME}/.zshrc" ;;
  */bash)
    if [ "$os" = darwin ]; then
      # macOS terminals open login shells, which read the first of these
      # that exists; creating .bash_profile would hide an existing .profile.
      rc="$HOME/.bash_profile"
      for f in "$HOME/.bash_profile" "$HOME/.bash_login" "$HOME/.profile"; do
        if [ -f "$f" ]; then rc="$f"; break; fi
      done
    else
      rc="$HOME/.bashrc"
    fi
    ;;
  */fish)
    rc="${XDG_CONFIG_HOME:-$HOME/.config}/fish/conf.d/quarter.fish"
    line="contains -- '$dir' \$PATH; or set -gx PATH '$dir' \$PATH"
    ;;
  */csh | */tcsh) by_hand ;;
  *)
    rc="$HOME/.profile"
    next="Log out and back in, then type quarter to start."
    ;;
esac

case "$rc" in
  "$HOME"/*) shown="~/${rc#"$HOME"/}" ;;
  *) shown="$rc" ;;
esac
if grep -qsF -- "$line" "$rc"; then
  say "$shown already puts $dir on your PATH."
else
  mkdir -p "${rc%/*}" 2>/dev/null || by_hand
  { printf '\n# Quarter (https://github.com/%s): its folder on the PATH\n%s\n' "$repo" "$line" >>"$rc"; } 2>/dev/null || by_hand
  say "Added $dir to your PATH in $shown."
fi
say "$next"
say "(Or start it now with: $dir/quarter)"
