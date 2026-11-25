狀態：草案／待批准

# ADR-014：品牌更名 NEXUS→ETHER 暨設計系統方向

- **狀態**：已決定（草案／待批准）
- **日期**：2026-11-25（Phase 00 品牌替換）
- **依據**：使用者 2026-11-25 00:42 下達的《ETHER — 完整品牌策略與識別企劃書》；使用者 2026-11-25 00:29 規格（品牌暫定名 NEXUS）；`docs/brand/BRAND_GUIDELINES.md`

## 問題

原始規格下達時的品牌暫定名 NEXUS 已由使用者品牌企劃書正式改為 ETHER，設計系統方向亦由「Material 3 深度客製」改為以 ETHER 品牌識別為準。既有 Phase 00 文件如何更名而不破壞架構決策？設計系統的準則為何？

## 決定

1. **品牌更名**：專案品牌一律為 ETHER（全名 Every Thread Holds Endless Remembrance，標語 One Connection. Endless Possibilities.，定位 A Human Connection Platform）。專案目錄由 `~/workspace/nexus/` 改為 `~/workspace/ether/`；全部文件內的 NEXUS／nexus 一律改為 ETHER／ether（標題、內文、錯誤碼前綴 `ETHER_`、Compose 服務名與網路名、環境變數預設值、路徑引用）。
2. **歷史保留**：`SPEC-SOURCE.md`（原始規格摘要）與 `ADR-005`（舊規格差異對照）內文保留原始寫法，僅加頂部追溯註記，說明 NEXUS 為暫定名、現已改為 ETHER；舊名僅供追溯，不得在其他文件出現。`~/workspace/messenger-app/` 舊路線文件不動。
3. **架構不變**：更名不改動任何架構決策——技術棧（Flutter＋Go＋PostgreSQL／Valkey／NATS／MinIO／Meilisearch）、22 模組邊界、表名、事件名、成本最佳化替換（ADR-009）皆維持不變。
4. **設計系統方向**：視覺方向為 Ethereal Modernism（空靈現代主義），以 `docs/brand/DESIGN_TOKENS.md` 為唯一色彩與字體來源；Material 3 僅為 Flutter 實作基底之一，不得以 Material 3 預設外觀代替品牌設計。原 PRD「Material 3 深度客製」表述已修正（見 PRD §1.4）。
5. **產品命名對照**：對外產品名依品牌架構（ETHER Chat／Voice／Meet／Space／AI／Moments／Memories／Connect／Studio／Business）；內部模組識別碼、表名、事件名不因產品命名而改（對照見 `docs/brand/BRAND_GUIDELINES.md` §11 與 `docs/architecture/MODULE_BOUNDARIES.md` 產品命名對照節）。
6. **品牌治理阻塞項**：商標近似檢索、主要市場商標類別評估、網域與社群帳號檢查、App Store／Google Play 名稱檢查，皆為品牌對外使用前阻塞項，全部未完成；未完成前不得對外使用 ETHER 品牌，不得宣稱品牌已可對外使用。Logo 與 App Icon 設計未實作，屬 Phase 02 工作。

## 後果

- 更名檔案：專案目錄 1 個、文件 30 份含 ETHER 引用；新增 `docs/brand/BRAND_GUIDELINES.md`、`docs/brand/DESIGN_TOKENS.md`、本 ADR。
- 交叉引用路徑已同步更新（`~/workspace/ether/`）；Compose YAML 重新解析通過（見品牌替換複驗）。
- Phase 02 設計系統須以 `DESIGN_TOKENS.md` 為準；未定稿的 token 不得編造。
