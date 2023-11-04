#!/usr/bin/env bash

# Define the GitHub repository and release API URL
repo="Lofter1/anyflip-downloader"
api_url="https://api.github.com/repos/$repo/releases/latest"
application_dir="$HOME/.local/share/anyflip-downloader"
install_dir="$HOME/.local/bin"

# Determine the platform (Linux, macOS, or Windows)
echo "Detect Platform and Architecture"
platform=""
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    platform="linux"
    architecture=$(uname -m)
elif [[ "$OSTYPE" == "darwin"* ]]; then
    platform="darwin"
    architecture=$(uname -m)
elif [[ "$OSTYPE" == "msys"* || "$OSTYPE" == "win32" ]]; then
    echo "Please use the Windows install script" 
else
    echo "Unsupported platform: $OSTYPE"
    exit 1
fi

if [["$architecture" == "x86_64"]]; then
  $architecture = "amd64"
fi

echo "Detected $platform $architecture"

latest_release=$(curl -s "$api_url")
grep_pattern="\"browser_download_url\": \"[^\"]*${platform}_${architecture}[^\"]*\""
download_url=$(echo "$latest_release" | grep -o "$grep_pattern" | cut -d '"' -f 4)
echo $download_url

echo "Found latest release at $download_url"

if [ -z "$download_url" ]; then
    echo "No release found for platform: $platform"
    exit 1
fi

# Downloading
temp_dir=$(mktemp -d)
temp_file="$temp_dir/anyflip-downloader.tar.gz"

echo "Download into $temp_file"
curl -L "$download_url" > "$temp_file"

# Create application and install dir if it doesn't already exist
mkdir -p "$application_dir"
mkdir -p "$install_dir"

# Installing
echo "Unpacking into $application_dir"
tar -zxvf "$temp_file" -C "$application_dir"
echo "Move binary into $install_dir"
mv "$application_dir/anyflip-downloader" "$install_dir"


# Add install dir to path if it is not already part of path
if [ ":$PATH:" != *"${install_dir}"* ]; then
  PATH="${install_dir}:${PATH}"
fi

# Cleanup
rm -rf "$temp_dir"
