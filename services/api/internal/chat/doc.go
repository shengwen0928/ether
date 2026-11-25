// Package chat 是 Chat（聊天室）模組的 Phase 05 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.6）：對話容器的建立、成員管理、
// 對話設定（背景、主題、通知、靜音、封存、置頂——設定表與相關端點
// 屬後續里程碑，本里程碑不實作）。一對一對話的建立須以 Friend
// 關係為前提，經 Friend 介面驗證，不得自行查好友表。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：conversations、
// conversation_members（conversation_settings 屬後續里程碑，見
// migrations/000006 檔頭）。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：
// User、Friend、Authentication（授權介面由中介層落實）。
// 禁止事項：不得寫入訊息本體（屬 Message）；不得刪除對話時連帶
// 物理刪除訊息（訊息 append-only 方向）；不得以對話成員身份代替
// Group／Community 的角色權限。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// 實作見 chat.go（領域＋記憶體儲存）、postgres.go（PostgreSQL
// 儲存庫）、handler.go（HTTP 端點）；儲存語義與 ERD 差異見
// migrations/000006_messaging.up.sql 檔頭。
package chat
