Set-Location C:\pwni-file-sync\release_build
# Windows 64-bit
mkdir omniblob-v1.0.0-windows-amd64
Copy-Item omniblob.exe omniblob-v1.0.0-windows-amd64\
Copy-Item README.md, MIGRATION_GUIDE.md, LICENSE omniblob-v1.0.0-windows-amd64\
Copy-Item configs omniblob-v1.0.0-windows-amd64\configs -Recurse
Compress-Archive -Path omniblob-v1.0.0-windows-amd64\* -DestinationPath C:\pwni-file-sync\omniblob-v1.0.0-windows-amd64.zip

# Windows 32-bit
mkdir omniblob-v1.0.0-windows-386
Copy-Item omniblob_32.exe omniblob-v1.0.0-windows-386\omniblob.exe
Copy-Item README.md, MIGRATION_GUIDE.md, LICENSE omniblob-v1.0.0-windows-386\
Copy-Item configs omniblob-v1.0.0-windows-386\configs -Recurse
Compress-Archive -Path omniblob-v1.0.0-windows-386\* -DestinationPath C:\pwni-file-sync\omniblob-v1.0.0-windows-386.zip

# Linux amd64
mkdir omniblob-v1.0.0-linux-amd64
Copy-Item omniblob_linux omniblob-v1.0.0-linux-amd64\omniblob
Copy-Item README.md, MIGRATION_GUIDE.md, LICENSE omniblob-v1.0.0-linux-amd64\
Copy-Item configs omniblob-v1.0.0-linux-amd64\configs -Recurse
Copy-Item systemd omniblob-v1.0.0-linux-amd64\systemd -Recurse
tar -czvf C:\pwni-file-sync\omniblob-v1.0.0-linux-amd64.tar.gz -C omniblob-v1.0.0-linux-amd64 .
