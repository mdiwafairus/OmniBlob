Set-Location C:\pwni-file-sync\release_build

# Overwrite configs in all folders
Copy-Item configs\config.yaml omniblob-v1.0.0-windows-amd64\configs\config.yaml -Force
Copy-Item configs\config.yaml omniblob-v1.0.0-windows-386\configs\config.yaml -Force
Copy-Item configs\config.yaml omniblob-v1.0.0-linux-amd64\configs\config.yaml -Force
Copy-Item configs\config.yaml omniblob-v1.0.0-linux-386\configs\config.yaml -Force

# Re-compress Windows 64
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-windows-amd64.zip -Force -ErrorAction SilentlyContinue
Compress-Archive -Path omniblob-v1.0.0-windows-amd64\* -DestinationPath C:\pwni-file-sync\omniblob-v1.0.0-windows-amd64.zip

# Re-compress Windows 32
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-windows-386.zip -Force -ErrorAction SilentlyContinue
Compress-Archive -Path omniblob-v1.0.0-windows-386\* -DestinationPath C:\pwni-file-sync\omniblob-v1.0.0-windows-386.zip

# Re-compress Linux 64
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-linux-amd64.tar.gz -Force -ErrorAction SilentlyContinue
tar -czvf C:\pwni-file-sync\omniblob-v1.0.0-linux-amd64.tar.gz -C omniblob-v1.0.0-linux-amd64 .

# Re-compress Linux 32
Remove-Item C:\pwni-file-sync\omniblob-v1.0.0-linux-386.tar.gz -Force -ErrorAction SilentlyContinue
tar -czvf C:\pwni-file-sync\omniblob-v1.0.0-linux-386.tar.gz -C omniblob-v1.0.0-linux-386 .

