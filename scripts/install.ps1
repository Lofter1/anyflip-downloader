# Define variables for the repository and tar.gz file name
$repoOwner = "elchinbaba"
$repoName = "anyflip-downloader"
$tarGzFileName = "anyflip-downloader.tar.gz"

# Use Get-WmiObject to query the Win32_Processor class
$processorInfo = $Env:PROCESSOR_ARCHITECTURE 

# Check the architecture and display a message accordingly
if ($processorInfo -eq "x86") {
    $architecture = "386"
}
elseif ($processorInfo -eq "AMD64") {
    $architecture = "amd64"
}
elseif ($processorInfo -eq "ARM64") {
    $architecture = "arm64"
}
else {
    Write-Host "Unable to determine the operating system architecture."
    exit
}
Write-Host "Detected architecture $architecture"


# Define the GitHub API URL to get the latest release information
$releaseUrl = "https://api.github.com/repos/$repoOwner/$repoName/releases/latest"

# Use Invoke-RestMethod to retrieve the latest release data
$latestRelease = Invoke-RestMethod -Uri $releaseUrl

# Get the download URL for the latest release asset (assuming it's a tar.gz file)
$downloadUrl = $latestRelease.assets | Where-Object { $_.name -like "*windows_$architecture.tar.gz" } | Select-Object -ExpandProperty browser_download_url

# Define the folder where you want to install the application
$installFolder = "$env:LocalAppData\anyflip-downloader"

# Create the installation folder if it doesn't exist
if (-not (Test-Path -Path $installFolder -PathType Container)) {
    New-Item -Path $installFolder -ItemType Directory -Force
}

# Define the path to save the downloaded tar.gz file
$tarGzFilePath = Join-Path -Path $installFolder -ChildPath $tarGzFileName

# Use Invoke-WebRequest to download the tar.gz file
Invoke-WebRequest -Uri $downloadUrl -OutFile $tarGzFilePath

# Expand the downloaded tar.gz file to the installation folder
tar -xzvf $tarGzFilePath -C $installFolder

# Clean up: remove the downloaded tar.gz file
Remove-Item -Path $tarGzFilePath -Force

# Optionally, add the installation folder to the system's PATH environment variable
# [System.Environment]::SetEnvironmentVariable('Path', $env:Path + ";$installFolder", [System.EnvironmentVariableTarget]::Machine)

# Display a message indicating the installation is complete
Write-Host "anyflip-downloader has been installed to $installFolder."
