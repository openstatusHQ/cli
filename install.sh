#!/bin/sh
# Copyright 2019-2025 the Deno authors. All rights reserved. MIT license.
# Copyright 2025 OpenStatus. All rights reserved. MIT license.
# Adopted from https://github.com/denoland/deno_install

set -e

if [ "$OS" = "Windows_NT" ]; then
	target="Windows_x86_64.zip"
else
	case $(uname -sm) in
	"Darwin arm64") target="Darwin_arm64.tar.gz" ;;
	"Darwin x86_64") target="Darwin_x86_64.tar.gz" ;;
	"Linux x86_64") target="Linux_x86_64.tar.gz" ;;
	"Linux aarch64") target="Linux_arm64.tar.gz" ;;
	*) target="unknown" ;;
	esac
fi

if [ "$target" = "unknown" ]; then
	echo "Note: openstatus is not supported on this platform"
	exit 0
fi

print_help_and_exit() {
	echo "Setup script for installing openstatus

Options:
  -y, --yes
    Skip interactive prompts and accept defaults
  --no-modify-path
    Don't add openstatus to the PATH environment variable
  -h, --help
    Print help
"
	echo "Note: openstatus was not installed"
	exit 0
}

get_latest_version() {
    curl --ssl-revoke-best-effort -s https://api.github.com/repos/openstatusHQ/cli/releases/latest | awk -F'"' '/"tag_name":/{print substr($4,1)}'
}

# Initialize variables
should_run_shell_setup=false
no_modify_path=false

# Simple arg parsing - look for help flag, otherwise
# ignore args starting with '-' and take the first
# positional arg as the deno version to install
for arg in "$@"; do
	case "$arg" in
	"-h")
		print_help_and_exit
		;;
	"--help")
		print_help_and_exit
		;;
	"-y")
		should_run_shell_setup=true
		;;
	"--yes")
		should_run_shell_setup=true
		;;
	"--no-modify-path")
		no_modify_path=true
		;;
	"-"*) ;;
	*)
		if [ -z "$openstatus_version" ]; then
			openstatus_version="$arg"
		fi
		;;
	esac
done

if [ -z "$openstatus_version" ]; then
	openstatus_version=$(get_latest_version)
fi

platform=$(echo "$target" | sed 's/\.\(tar\.gz\|zip\)$//')
echo "Installing openstatus ${openstatus_version} for ${platform}"

openstatus_uri="https://github.com/openstatusHQ/cli/releases/download/${openstatus_version}/cli_${target}"
openstatus_install="${OPENSTATUS_INSTALL:-$HOME/.openstatus}"
bin_dir="$openstatus_install/bin"
exe="$bin_dir/openstatus"

echo "Downloading openstatus from $openstatus_uri"
if [ ! -d "$bin_dir" ]; then
	mkdir -p "$bin_dir"
fi

# Download and extract the archive
tmp_dir=$(mktemp -d)
trap "rm -rf $tmp_dir" EXIT

if echo "$target" | grep -q "\.zip$"; then
	# Windows zip file
	curl --fail --location --progress-bar --output "$tmp_dir/openstatus.zip" "$openstatus_uri"
	unzip -q "$tmp_dir/openstatus.zip" -d "$tmp_dir"
	mv "$tmp_dir/openstatus.exe" "$exe"
else
	# Unix tar.gz file
	curl --fail --location --progress-bar --output "$tmp_dir/openstatus.tar.gz" "$openstatus_uri"
	tar -xzf "$tmp_dir/openstatus.tar.gz" -C "$tmp_dir"
	mv "$tmp_dir/openstatus" "$exe"
fi

chmod +x "$exe"

echo "openstatus was installed successfully to $exe"

run_shell_setup() {
	local rc_files=""
	local current_shell=""

	# Try to detect the current shell more reliably
	if [ -n "$SHELL" ]; then
		current_shell=$(basename "$SHELL")
	elif [ -n "$ZSH_VERSION" ]; then
		current_shell="zsh"
	elif [ -n "$BASH_VERSION" ]; then
		current_shell="bash"
	elif [ -n "$KSH_VERSION" ]; then
		current_shell="ksh"
	elif [ -n "$FISH_VERSION" ]; then
		current_shell="fish"
	else
		current_shell="sh"
	fi

	# Determine which rc files to modify based on shell
	case "$current_shell" in
		zsh)
			rc_files="$HOME/.zshrc"
			;;
		bash)
			rc_files="$HOME/.bashrc"
			# Add .bash_profile for login shells on macOS
			if [ "$(uname -s)" = "Darwin" ]; then
				rc_files="$rc_files $HOME/.bash_profile"
			fi
			;;
		fish)
			# Fish has a different way of setting PATH
			mkdir -p "$HOME/.config/fish/conf.d"
			echo "set -gx OPENSTATUS_INSTALL \"$openstatus_install\"" > "$HOME/.config/fish/conf.d/openstatus.fish"
			echo "set -gx PATH \$OPENSTATUS_INSTALL/bin \$PATH" >> "$HOME/.config/fish/conf.d/openstatus.fish"
			echo "Added openstatus to PATH in fish configuration"
			return
			;;
		*)
			# Default to .profile for other shells
			rc_files="$HOME/.profile"
			;;
	esac

	# Add setup line to each rc file
	for rc_file in $rc_files; do
		if [ ! -f "$rc_file" ]; then
			touch "$rc_file"
		fi

		if ! grep -q "$openstatus_install/bin" "$rc_file"; then
			echo "" >> "$rc_file"
			echo "# openstatus setup" >> "$rc_file"
			echo "export OPENSTATUS_INSTALL=\"$openstatus_install\"" >> "$rc_file"
			echo "export PATH=\"\$OPENSTATUS_INSTALL/bin:\$PATH\"" >> "$rc_file"
			echo "Added openstatus to PATH in $rc_file"
		else
			echo "openstatus already in PATH in $rc_file"
		fi
	done

	echo "Restart your shell or run 'source $rc_file' to use openstatus"
}

# Add openstatus to PATH for non-Windows if needed
if [ "$OS" != "Windows_NT" ] && [ "$no_modify_path" = false ]; then
    # If not automatic setup, but interactive is possible, ask user
    if [ "$should_run_shell_setup" = false ] && [ -t 0 ]; then
        echo ""
        echo "Do you want to add openstatus to your PATH? [y/N]"
        read -r answer
        if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
            should_run_shell_setup=true
        fi
    fi

    if [ "$should_run_shell_setup" = true ]; then
        run_shell_setup
    else
        echo ""
        echo "To manually add openstatus to your path:"
        echo "  export OPENSTATUS_INSTALL=\"$openstatus_install\""
        echo "  export PATH=\"\$OPENSTATUS_INSTALL/bin:\$PATH\""
        echo ""
        echo "To do this automatically in the future, run with -y or --yes"
    fi
fi

if command -v openstatus >/dev/null; then
	echo "Run 'openstatus --help' to get started"
else
	echo "Run '$exe --help' to get started"
fi
echo
echo "Stuck? Join our Discord https://openstatus.dev/discord"
