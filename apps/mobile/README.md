# ETHER Mobile（Flutter）— Phase 01 骨架

狀態：骨架，待 CI 驗證。本機 VM 無 flutter，建置、分析、測試皆未驗證，不得宣稱為已驗證。

## 這是什麼

Flutter Mobile＋Desktop 骨架（Android／iOS／Windows／macOS 共用 Flutter 核心，見
`docs/architecture/SYSTEM_ARCHITECTURE.md` §3.3）。本階段只有最小可啟動殼：
一個佔位頁、一個佔位路由、8 個品牌色、佔位設定與註解型 stub。尚無任何功能。

## 前置工具鏈

- Flutter SDK：版本約束見 `pubspec.yaml`（`flutter: ">=3.24.0"`，Dart SDK `>=3.5.0 <4.0.0`）。
  確切 SDK 版本由 CI 定案並固定；本機 VM 無 flutter，須在 CI 或有 Flutter 的環境執行。
- iOS 建置須走 macOS CI（見 `PLATFORM_RUNNERS.md` 與 ADR-003）；Linux 上不可能建置 iOS。
- 平台 runner 目錄須由 `flutter create` 在 CI 產生，不得手工編造（見 `PLATFORM_RUNNERS.md`）。

## 建置與檢查（皆待 CI 驗證）

```sh
cd apps/mobile
flutter pub get
flutter analyze
flutter test
flutter run
```

以上指令在本機 VM 未執行過（無 flutter）；CI 須逐項回報結果。`flutter run` 預期只顯示
佔位頁文字「Phase 01 骨架，尚無功能」，無按鈕、無功能。

## 檔案結構

- `pubspec.yaml`：依賴全部固定版本（固定於 2026-11-25，未經 pub 解析驗證）。
- `analysis_options.yaml`：嚴格 lint 設定。
- `lib/main.dart`：最小可啟動骨架（Material 3、Riverpod、GoRouter 單一佔位路由）。
- `lib/core/config/`：環境設定佔位（全部佔位符，無真實金鑰／URL）。
- `lib/core/tokens/`：`app_colors.dart`（只寫 8 個已定品牌色）。
- `lib/core/error/`：錯誤處理契約註解佔位（契約見 `docs/api/ERROR_CODES.md`）。
- `lib/core/database/`：Drift 註解型 stub（建置延至 Phase 05）。
- `lib/core/storage/`：安全儲存佔位說明（用途延至 Phase 03）。
- `lib/common/`：共用目錄佔位（依 SPEC §26 保留位置）。
- `lib/design_system/`：README 佔位說明（元件庫是 Phase 02 工作）。
- `lib/features/`：12 個 feature 空目錄＋對照 README（全部空骨架）。
- `test/`：一個 placeholder widget test（skip，TODO，不得冒充測試通過）。
- `PLATFORM_RUNNERS.md`：平台 runner 產生方式與 iOS macOS CI 說明。

## 目前狀態

- 骨架檔案已建立，僅做靜態檢查（見 `docs/phase01/flutter-notes.md`）。
- pubspec 解析、編譯、分析、測試，全部待 CI 驗證。

## 缺口清單

1. 平台 runner 目錄未產生（須 `flutter create`，參數未定案）。
2. 依賴版本未經 pub 解析驗證（固定於 2026-11-25，見 `docs/phase01/flutter-notes.md`）。
3. Design Tokens 除 8 個品牌色外全部待 Phase 02 定稿；元件庫、Logo、App Icon PNG 匯出皆為 Phase 02。
4. Drift 資料庫建置延至 Phase 05；`flutter_secure_storage` 呼叫延至 Phase 03。
5. 環境設定注入方式未定案；Keycloak／API／LiveKit 端點未定案。
6. 錯誤處理、全部 feature 功能，皆待其負責階段。
7. iOS 建置、簽名、上架憑證，皆為阻塞項（見 ADR-003／ADR-004）。

## 禁止事項

- 不得宣稱未驗證的事；不得寫假資料、Mock 後端、無作用按鈕。
- 不得把憑證、金鑰、E2EE 明文寫進程式或文件。
- 不得建立 `.github/` 目錄、不得 push、不得建 repo／PR／Release；成果只留本機。
