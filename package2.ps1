Set-Location C:\pwni-file-sync\release_build

# Re-copy updated README to existing folders
Copy-Item README.md omniblob-v1.0.0-windows-amd64\
Copy-Item README.md omniblob-v1.0.0-windows-386\
Copy-Item README.md omniblob-v1.0.0-linux-amd64\

# Re-compress Windows 64
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-windows-amd64.zip -Force -ErrorAction SilentlyContinue
Compress-Archive -Path omniblob-v1.0.0-windows-amd64\* -DestinationPath C:\pwni-file-sync\omniblob-v1.0.0-windows-amd64.zip

# Re-compress Windows 32
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-windows-386.zip -Force -ErrorAction SilentlyContinue
Compress-Archive -Path omniblob-v1.0.0-windows-386\* -DestinationPath C:\pwni-file-sync\omniblob-v1.0.0-windows-386.zip

# Re-compress Linux 64
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-linux-amd64.tar.gz -Force -ErrorAction SilentlyContinue
tar -czvf C:\pwni-file-sync\omniblob-v1.0.0-linux-amd64.tar.gz -C omniblob-v1.0.0-linux-amd64 .

# Linux 32-bit (Wait for build if running)
while (Get-Process -Name go -ErrorAction SilentlyContinue) { Start-Sleep -Seconds 2 }

mkdir omniblob-v1.0.0-linux-386 -ErrorAction SilentlyContinue
Copy-Item omniblob_linux_32 omniblob-v1.0.0-linux-386\omniblob
Copy-Item README.md, MIGRATION_GUIDE.md, LICENSE omniblob-v1.0.0-linux-386\
Copy-Item configs omniblob-v1.0.0-linux-386\configs -Recurse
Copy-Item systemd omniblob-v1.0.0-linux-386\systemd -Recurse
tar -czvf C:\pwni-file-sync\omniblob-v1.0.0-linux-386.tar.gz -C omniblob-v1.0.0-linux-386 .

