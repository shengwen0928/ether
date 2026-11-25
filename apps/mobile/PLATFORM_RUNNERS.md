# 平台 Runner 目錄 — Phase 01 說明

狀態：Phase 01 骨架不含任何平台 runner 目錄。本文件說明 runner 目錄的產生方式，不得手工編造。

## 規則

- `android/`、`ios/`、`windows/`、`macos/`、`linux/` runner 目錄，須由 `flutter create`
  在 CI 產生，不得手工編造、不得複製其他專案的 runner 目錄。
- 產生指令的參數（組織識別碼／Bundle ID、應用程式名稱等）尚未定案，不得編造；
  待協調員／Phase 01 審查定案後，由 CI 以定案參數執行。
- 本機 VM 無 flutter，無法在本機執行 `flutter create`；產生與驗證皆待 CI 驗證。

## iOS 須走 macOS CI

- 依據：`docs/decisions/ADR-003-iOS須走macOS-CI.md`、`SPEC-SOURCE.md` 環境實情。
- iOS 建置須 Xcode，Xcode 只在 macOS 上運行；Linux 上不可能建置 iOS。
- Flutter iOS 產物（Archive、上架包）的建置、簽名與驗證，皆在 macOS CI 環境執行；
  不得以 Android 建置通過代替 iOS 驗證。
- Apple Developer 帳號、上架憑證與簽名設定為阻塞項（見 ADR-004）：未取得前，CI 流程
  只寫介面與設定範本（無真憑證），不得假裝已可上架。簽名金鑰與憑證一律經 CI Secrets／
  OpenBao 注入，不得入 Git。

## 桌面平台

- 桌面目標為 Windows／macOS（見 `docs/PLATFORM_MATRIX.md` §1.2：多欄、側欄、快捷鍵、
  拖放、托盤）；桌面介面不得複製手機畫面，屬 Phase 10 工作。
- Windows／macOS 安裝包格式與簽名／公證流程，待 Phase 11 發布階段定案，成本待查證。

## 缺口

- `flutter create` 參數（組織識別碼、Bundle ID）未定案。
- macOS CI runner 的成本待查證，屬仍可能付費／額度項目，不得宣稱免費。
