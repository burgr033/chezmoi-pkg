# chezmoi-pkg

i have a really quirky setup where i use a .chezmoidata/machine.toml file to manage machine specific package installation (and tracking) on my machines which then gets installed during `chezmoi apply` with the help of a script (see below)

This kinda achieves "declarative" package management on arch with the help of chezmoi

## usage

```bash
go install github.com/burgr033/chezmoi-pkg
#---
chezmoi pkg add thefuck
chezmoi pkg remove thefuck
chezmoi pkg list
```
## Config

This tool just has a config file under ~/.config/chezmoi-pkg/pkg.yaml

```yaml
file: $HOME/.local/chezmoi/.chezmoidata/machine.toml
```

## generated file

will look something like this:

```toml
[packages]
[packages.linux]
[packages.linux.arch]
[packages.linux.arch.HOSTNAME]
packages = ['amd-ucode', 'base', 'btrfs-assistant', 'btrfs-progs', 'cups', 'dosfstools', 'efibootmgr', 'inotify-tools', 'linux', 'linux-firmware', 'linux-headers', 'linux-lts', 'linux-lts-headers', 'linux-zen', 'linux-zen-headers', 'networkmanager', 'ntfs-3g', 'snapper', 'sudo', 'syslinux', 'vulkan-radeon', 'xf86-video-amdgpu', 'xf86-video-ati', 'xorg-server', 'xorg-xhost', 'xorg-xinit', 'xorg-xrandr', 'yay-bin', 'zram-generator']
```

## usage

Basically you need a template in chezmoi which installs your packages on apply.

Example: (this example also incorporates my profiles (office, common, desktop, etc.) from packages.toml in my chezmoi source path.

```tmpl
#!/bin/bash
set -eufo pipefail

# --- 1. Data Gathering (Go Template) ---
{{- $combinedPackages := list -}}
{{- range .profiles -}}
  {{- $profile := . -}}
  {{- with (index $.packages.linux.arch $profile) -}}
    {{- if .packages -}}
      {{- $combinedPackages = concat $combinedPackages .packages -}}
    {{- end -}}
  {{- end -}}
{{- end -}}

{{with (index $.packages.linux.arch .chezmoi.hostname)}}
{{- if .packages }}
      {{- $combinedPackages = concat $combinedPackages .packages -}}
{{- end }}
{{end}}


# --- 3. Normalization ---
TARGET_LIST=$(cat <<EOF | LC_ALL=C sort -u
{{ $combinedPackages | join "\n" }}
EOF
)

CURRENT_LIST=$(yay -Qeq | LC_ALL=C sort -u)

# --- 4. Delta Calculation ---
MISSING=$(LC_ALL=C comm -23 <(echo "$TARGET_LIST") <(echo "$CURRENT_LIST") | sed '/^$/d')

# Initial "Extra" calculation
RAW_EXTRA=$(LC_ALL=C comm -13 <(echo "$TARGET_LIST") <(echo "$CURRENT_LIST") | sed '/^$/d')

# Filter out whitelisted items from the EXTRA list
EXTRA=$(echo "$RAW_EXTRA")

# --- 5. The Fancy Dashboard View ---
echo -e "\n\033[1;34mPackage Synchronization\033[0m"
echo -e "------------------------------------------"

if [ -z "$MISSING" ] && [ -z "$EXTRA" ]; then
    echo -e "\033[0;32m✓ System is in sync.\033[0m"
    exit 0
fi

{
    if [ -n "$MISSING" ]; then
        while read -r pkg; do
            echo -e "\033[0;32m[+]\033[0m\t$pkg\t(Install)"
        done <<< "$MISSING"
    fi

    if [ -n "$EXTRA" ]; then
        while read -r pkg; do
            echo -e "\033[0;31m[-]\033[0m\t$pkg\t(Remove)"
        done <<< "$EXTRA"
    fi
} | column -t -s $'\t'

echo -e "------------------------------------------\n"

# --- 6. Execution ---
echo -ne "\033[1;32m➜ sync packages? [y/N]: \033[0m"
read -r confirm_in </dev/tty
if [[ "$confirm_in" =~ ^[Yy]$ ]]; then
    echo "$EXTRA" | xargs -r yay -Rs --noconfirm
    echo "$MISSING" | xargs -r yay -S --needed --noconfirm
fi
```

So everytime i add or remove a package it runs chezmoi sync and checks the diff. installs and uninstalls accordingly
