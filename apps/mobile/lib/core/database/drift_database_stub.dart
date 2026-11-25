// ETHER Phase 01 骨架 — Drift 資料庫 stub。
//
// 本檔案僅為註解型 stub，無 Database 類別、無 schema、無 DAO。
// Drift 資料庫建置延至 Phase 05（即時訊息核心，手冊 G）；
// 本階段不得寫無法編譯的假 Database 類別，不得編造 schema。
//
// 依據與待定事項：
// - 資料表名以 SPEC-SOURCE.md §20 與 docs/database/ERD.md 為準，不得改名、不得預寫欄位。
// - codegen（drift_dev／build_runner）在 CI 執行；本機 VM 無 flutter，codegen 未驗證，待 CI 驗證。
// - 原生 SQLite 驅動接線方式、Migration 策略，待 Phase 05 定稿。
// - E2EE 相關本機資料的加密儲存方式，待安全性架構與 E2EE 文件定稿，不得自行設計。
//
// TODO(Phase 05): 建立 Drift 資料庫、資料表定義與 Migration，並以真實裝置驗收離線／重連恢復。
