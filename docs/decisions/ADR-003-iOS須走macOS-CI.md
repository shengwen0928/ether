狀態：草案／待批准

# ADR-003：iOS 須走 macOS CI（Linux 無法建置）

- **狀態**：已決定（草案／待批准）
- **日期**：2026-11-25（Phase 00）
- **依據**：SPEC-SOURCE.md「環境實情」、ADR-008

## 問題

iOS App（Flutter）應如何建置？本機 Linux VM 能否建置 iOS？

## 選項

| 選項 | 可行性 |
|---|---|
| A. 本機 Linux VM 建置 iOS | 不可行：iOS 建置須 Xcode，Xcode 只在 macOS 上運行；Linux 上不可能 |
| B. macOS CI runner 建置 iOS | 可行：GitHub Actions 的 macOS runner（或自架 macOS 主機）可建置 iOS 專案、Archive 與上架產物 |
| C. 不建置 iOS，只交付 Android | 不可行：SPEC-SOURCE.md 要求五平台（Android、iOS、Windows、macOS、Web），最終交付含「iOS 可建置專案」，不得刪減 |

## 決定

選 B：iOS 建置一律走 macOS CI。Flutter iOS 產物（Archive、上架包）的建置、簽名與驗證，皆在 macOS CI 環境執行；本機 Linux VM 不嘗試建置 iOS，不得以「Linux 上編不過」為由宣稱 iOS 不可行，也不得以 Android 建置通過代替 iOS 驗證。

## 理由

1. 環境實測（2026-11-25）：本機 Linux VM 無 flutter、無 go、無 docker；iOS 建置在 Linux 上不可能，須走 macOS CI runner（SPEC-SOURCE.md 原文）。
2. Apple Developer 帳號、上架憑證與簽名設定為阻塞項（見 ADR-004）：未取得前，CI 流程只寫介面與設定範本（無真憑證），不得假裝已可上架。
3. 簽名金鑰與憑證一律經 CI Secrets／OpenBao 注入，不得入 Git（見 `SECURITY_ARCHITECTURE.md` §6）。

## 後果

- Phase 01 的 CI 須含 macOS 作業（至少：Flutter iOS 可建置檢查）；macOS CI 的成本（runner 費用）待查證，屬仍可能付費／額度項目，不得宣稱免費；不得代使用者決定花錢或綁卡。
- iOS 實機驗證須在真實 iPhone 上進行，不得以模擬器或 Android 實機代替。
