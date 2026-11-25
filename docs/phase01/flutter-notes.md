# Flutter 軌道筆記 — Phase 01 骨架（docs/phase01/flutter-notes.md）

撰寫日：2026-11-25。軌道：Flutter（apps/mobile）。本機 VM 無 flutter，一切建置驗證待 CI。

## 1. 檔案清單（新增 15 個檔案＋12 個空 feature 目錄）

| 路徑 | 用途 |
|---|---|
| `apps/mobile/pubspec.yaml` | 依賴全部固定版本，檔頭註明固定於 2026-11-25、未經 pub 解析驗證 |
| `apps/mobile/analysis_options.yaml` | 嚴格 lint 設定（flutter_lints 固定版本） |
| `apps/mobile/lib/main.dart` | 最小可啟動骨架：Material 3、Riverpod、GoRouter 單一佔位路由 |
| `apps/mobile/lib/core/config/app_config.dart` | 環境設定佔位（全部佔位符，無真實金鑰／URL） |
| `apps/mobile/lib/core/tokens/app_colors.dart` | 只寫 8 個已定品牌色；其餘 token 一律 TODO |
| `apps/mobile/lib/core/error/error_contract.dart` | 註解型錯誤契約佔位（契約見 docs/api/ERROR_CODES.md） |
| `apps/mobile/lib/core/database/drift_database_stub.dart` | 註解型 Drift stub（建置延至 Phase 05，無假 Database 類別） |
| `apps/mobile/lib/core/storage/README.md` | 安全儲存佔位說明（用途延至 Phase 03，本階段不呼叫） |
| `apps/mobile/lib/common/README.md` | 共用目錄佔位（依 SPEC §26 保留位置） |
| `apps/mobile/lib/design_system/README.md` | 元件庫佔位說明（Phase 02 工作，不建元件、不寫死色碼） |
| `apps/mobile/lib/features/README.md` | 12 個 feature 對照表（全部 Phase 01：空骨架） |
| `apps/mobile/lib/features/{auth,chat,contacts,friends,groups,communities,calls,timeline,stickers,notifications,settings,navigation}/` | 空目錄，內無任何檔案／程式碼 |
| `apps/mobile/PLATFORM_RUNNERS.md` | Runner 目錄須由 `flutter create` 在 CI 產生；iOS 走 macOS CI（ADR-003） |
| `apps/mobile/README.md` | 建置方式、狀態、缺口清單 |
| `apps/mobile/test/placeholder_test.dart` | Placeholder widget test（skip: true，無假斷言） |

未修改任何 Phase 00 文件（`SPEC-SOURCE.md`、`docs/` 下既有文件、`docker-compose.yml`、
`.env.example` 皆未動）；未建立 `.github/`；未 push、未建 repo／PR／Release；未碰
`~/workspace/messenger-app/`。

## 2. 版本固定依據

- 固定版本（無 `^` 浮動寫法）：`flutter_riverpod 2.5.1`、`go_router 13.2.0`、`dio 5.7.0`、
  `drift 2.18.0`、`flutter_secure_storage 9.2.2`；dev：`flutter_lints 4.0.0`、
  `drift_dev 2.18.0`、`build_runner 2.4.13`。
- 依據：撰寫當日（2026-11-25）所知的穩定版。未經 pub.dev 即時查證、未經 `flutter pub get`
  解析，待 CI 驗證。若 CI 解析失敗，由協調員調整固定版本，不得改為浮動寫法、不得宣稱已驗證。
- SDK 約束：Dart `>=3.5.0 <4.0.0`、Flutter `>=3.24.0`（見 pubspec.yaml）。
- 依賴數量精簡：只列骨架必需。Drift 原生驅動（如 `sqlite3_flutter_libs`）未列入，
  接線方式待 Phase 05 定稿；WebRTC／FCM／APNs 等後階段依賴一律未列入。

## 3. 靜態驗證結果（本機可做的檢查，皆已執行）

1. YAML 格式：`pubspec.yaml`、`analysis_options.yaml` 以 Python `yaml.safe_load` 解析通過；
   依賴／dev 依賴鍵值與固定版本逐項核對無誤。
2. 色碼逐字核對：`app_colors.dart` 的 8 個 `0xFF……` 值，與 `DESIGN_TOKENS.md` §1.1 的
   8 個色碼（#101923、#182A3A、#86B9D2、#C7D5DE、#F4F7F9、#65D4D6、#A99ACB、#D9B98C）
   集合完全一致；全專案 Dart 檔案中除 `app_colors.dart` 外無任何色碼字面值；
   `main.dart` 無 `Color(…)` 字面值，只引用 `AppColors`。
3. 檔案引用一致性：`main.dart` 的相對 import 目標 `core/tokens/app_colors.dart` 存在；
   12 個 feature 目錄皆存在且為空；`design_system/`、`common/` 只有 README 佔位。
4. 括號／大括號平衡：6 個 Dart 檔案逐檔計數，`{}`、`()`、`[]` 皆平衡。
5. 禁止項掃描：無 `Button` 元件、無 `expect(` 假斷言、測試含 `skip: true`；
   無 `messenger-app` 字串；無 `.github` 目錄（`apps/mobile/README.md` 僅在禁止事項句中提及
   `.github/` 字樣，非目錄）；`ether/` 下無 `.git`。
6. Secrets 掃描：以憑證賦值模式（password／api key／secret／token 值、Bearer）掃描新增檔案，
   無憑證值命中；`app_config.dart` 只有空字串佔位與 `String.fromEnvironment` 讀取，無真實 URL。
7. 未驗證（誠實標示）：`flutter pub get`、`flutter analyze`、`flutter test`、
   `flutter run`、Drift codegen，皆因本機無 flutter 而未執行，待 CI 驗證。

## 4. 已知缺口

1. 平台 runner 目錄未產生（`flutter create` 參數未定案，見 `PLATFORM_RUNNERS.md`）。
2. 依賴版本未經 pub 解析驗證（見 §2）。
3. `silverMist`、`warmHorizon` 已定義但尚未被 `main.dart` 主題引用（待 Phase 02 token 定稿後使用，
   不得為用而用編造語義）。
4. Design Tokens 除 8 色外全部待 Phase 02；元件庫、Logo、App Icon PNG 匯出皆為 Phase 02。
5. Drift 建置延至 Phase 05；安全儲存呼叫延至 Phase 03；環境注入方式未定案。
6. 錯誤處理實作、全部 feature 功能，待其負責階段（見 `lib/features/README.md`）。

## 5. 阻塞項（需協調員／使用者決定，非本軌道可自行決定）

1. CI 環境：本機 VM 無 flutter／go／docker；`flutter create`、pub 解析、編譯、測試須走 CI；
   iOS 須走 macOS CI（ADR-003），macOS runner 成本待查證。
2. `flutter create` 參數（組織識別碼／Bundle ID）未定案，不得編造。
3. Keycloak 配置定稿（Phase 03 阻塞項，見 ROADMAP.md）；Google／Apple OAuth 憑證未取得，
   不得當已完成整合。
4. 品牌對外使用前檢查（商標／網域／商店名稱）全部未完成（見 BRAND_GUIDELINES.md §17）；
   本階段不得宣稱 launcher 圖示已完成。
