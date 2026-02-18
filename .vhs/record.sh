#!/usr/bin/env bash
set -euo pipefail

# Local VHS recording wrapper — creates an isolated temp HOME so real
# dotfiles are never touched, seeds chezmoi state from testdata/dotfiles/,
# then runs VHS tape(s) from the project root.

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMPHOME=""

cleanup() {
  if [ -n "$TMPHOME" ] && [ -d "$TMPHOME" ]; then
    rm -rf "$TMPHOME"
  fi
}
trap cleanup EXIT

# --- Create isolated HOME ---
TMPHOME="$(mktemp -d)"
export HOME="$TMPHOME"
export XDG_CONFIG_HOME="$TMPHOME/.config"
export XDG_DATA_HOME="$TMPHOME/.local/share"
export COLORTERM=truecolor

# Use chezit binary from bin/ (built by `task record` dep)
export PATH="$PROJECT_ROOT/bin:$PATH"

# --- Seed chezmoi source state ---
mkdir -p "$HOME/.local/share/chezmoi"
cp -r "$PROJECT_ROOT/testdata/dotfiles/." "$HOME/.local/share/chezmoi/"

# Ensure chezmoi scripts are executable
chmod +x "$HOME/.local/share/chezmoi/.chezmoiscripts/"*

# HOME is redirected to TMPHOME above — chezmoi operates entirely in the temp dir.
chezmoi init --apply

# --- Configure git and create initial commit ---
# chezmoi init already runs git init; we just need identity + first commit.
cd "$HOME/.local/share/chezmoi"
git config user.email "demo@example.com"
git config user.name "Demo User"
git add -A
git commit -q -m "initial dotfiles"

# --- Create local drift (target differs from source — shows as drift) ---
# 1. zshrc: append a TODO comment
echo "" >> "$HOME/.zshrc"
echo "# TODO: add cargo to PATH" >> "$HOME/.zshrc"

# 2. ghostty: change font size (14 → 16)
sed -i '' 's/font-size = 14/font-size = 16/' "$HOME/.config/ghostty/config"

# --- Create unstaged source changes ---
# 1. ripgreprc: add a new glob exclusion
echo "--glob=!dist" >> dot_config/ripgrep/ripgreprc

# 2. starship: add git_status section
printf '\n[git_status]\nformat = "[$all_status]($style) "\n' >> dot_config/starship.toml

# 3. ssh config: add a new host
printf '\nHost work\n    HostName work.example.com\n    User deploy\n    IdentityFile ~/.ssh/id_work\n' >> dot_ssh/private_config

# --- Create staged source changes ---
# 1. bashrc: add a new alias
echo "alias la='ls -A'" >> dot_bashrc
git add dot_bashrc

# 2. brew.env: add analytics opt-out
echo "HOMEBREW_NO_ANALYTICS=1" >> dot_config/homebrew/brew.env.tmpl
git add dot_config/homebrew/brew.env.tmpl

# 3. zprofile: add rust PATH
echo 'export PATH="$HOME/.cargo/bin:$PATH"' >> dot_zprofile
git add dot_zprofile

# --- Run VHS from project root ---
cd "$PROJECT_ROOT"
mkdir -p docs/assets

if [ $# -eq 0 ]; then
  for tape in .vhs/*.tape; do
    [ "$(basename "$tape")" = "_settings.tape" ] && continue
    echo "Recording $(basename "$tape" .tape)..."
    vhs "$tape"
  done
else
  for name in "$@"; do
    echo "Recording ${name}..."
    vhs ".vhs/${name}.tape"
  done
fi
