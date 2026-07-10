APP_NAME ?= Excel图片链接替换工具
APP_ID ?= com.bestfulfill.excelimgreplacer
VERSION ?= dev
CMD_PATH ?= ./cmd/desktop
ICON_PATH ?= $(CURDIR)/icon.png
DIST_DIR ?= dist
BUILD_DIR ?= build

MAC_ARCH := $(shell uname -m)
MAC_APP := $(BUILD_DIR)/$(APP_NAME).app
MAC_DMG := $(DIST_DIR)/$(APP_NAME)-$(VERSION)-macos-$(MAC_ARCH).dmg

WIN_BUILD_NAME ?= ExcelImageReplacer
WIN_EXE := $(DIST_DIR)/$(WIN_BUILD_NAME)-$(VERSION)-windows-amd64.exe

APPLE_SIGN_IDENTITY ?=
NOTARY_PROFILE ?=

WINDOWS_CERT_PATH ?=
WINDOWS_CERT_PASSWORD ?=
WINDOWS_TIMESTAMP_URL ?= http://timestamp.digicert.com

.PHONY: help test vet build clean prepare package-macos package-macos-signed macos-app macos-sign-app macos-dmg macos-notarize package-windows sign-windows

help:
	@echo "Targets:"
	@echo "  make test                 Run Go tests"
	@echo "  make vet                  Run go vet"
	@echo "  make build                Build local macOS binary"
	@echo "  make package-macos        Build unsigned macOS .dmg"
	@echo "  make package-macos-signed Build signed and notarized macOS .dmg"
	@echo "  make package-windows      Build Windows .exe on a Windows host"
	@echo "  make sign-windows         Sign Windows .exe on a Windows host"
	@echo ""
	@echo "Signing variables:"
	@echo "  APPLE_SIGN_IDENTITY='Developer ID Application: Company (TEAMID)'"
	@echo "  NOTARY_PROFILE='notarytool-profile'"
	@echo "  WINDOWS_CERT_PATH='C:/path/cert.pfx'"
	@echo "  WINDOWS_CERT_PASSWORD='password'"

test:
	go test ./...

vet:
	go vet ./...

build: prepare
	go build -o "$(DIST_DIR)/$(WIN_BUILD_NAME)" "$(CMD_PATH)"

clean:
	rm -rf "$(BUILD_DIR)" "$(DIST_DIR)"

prepare:
	mkdir -p "$(BUILD_DIR)" "$(DIST_DIR)"

package-macos: clean prepare macos-app macos-dmg
	@echo "Created $(MAC_DMG)"

package-macos-signed: clean prepare macos-app macos-sign-app macos-dmg macos-notarize
	@echo "Created signed and notarized $(MAC_DMG)"

macos-app:
	cd "$(BUILD_DIR)" && fyne package --app-id "$(APP_ID)" --icon "$(ICON_PATH)" --name "$(APP_NAME)" -src "../$(CMD_PATH)"

macos-sign-app:
	@test -n "$(APPLE_SIGN_IDENTITY)" || (echo "Set APPLE_SIGN_IDENTITY first"; exit 1)
	codesign --force --deep --options runtime --timestamp --sign "$(APPLE_SIGN_IDENTITY)" "$(MAC_APP)"
	codesign --verify --deep --strict --verbose=2 "$(MAC_APP)"

macos-dmg:
	hdiutil create -volname "$(APP_NAME)" -srcfolder "$(MAC_APP)" -ov -format UDZO "$(MAC_DMG)"

macos-notarize:
	@test -n "$(NOTARY_PROFILE)" || (echo "Set NOTARY_PROFILE first"; exit 1)
	xcrun notarytool submit "$(MAC_DMG)" --keychain-profile "$(NOTARY_PROFILE)" --wait
	xcrun stapler staple "$(MAC_DMG)"

package-windows: prepare
	@test "$$(go env GOOS)" = "windows" || (echo "package-windows must run on a Windows host"; exit 1)
	fyne package --os windows --app-id "$(APP_ID)" --icon "$(ICON_PATH)" --name "$(WIN_BUILD_NAME)" -src "$(CMD_PATH)" --release
	mv "$(WIN_BUILD_NAME).exe" "$(WIN_EXE)"
	@echo "Created $(WIN_EXE)"

sign-windows:
	@test -n "$(WINDOWS_CERT_PATH)" || (echo "Set WINDOWS_CERT_PATH first"; exit 1)
	@test -n "$(WINDOWS_CERT_PASSWORD)" || (echo "Set WINDOWS_CERT_PASSWORD first"; exit 1)
	signtool sign /fd SHA256 /td SHA256 /tr "$(WINDOWS_TIMESTAMP_URL)" /f "$(WINDOWS_CERT_PATH)" /p "$(WINDOWS_CERT_PASSWORD)" "$(WIN_EXE)"
