// Package bot 是 Bot（機器人） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：Bot 應用建立、Token 核發與撤銷、Webhook 註冊、事件訂閱、收／發訊息的 Bot 側、群組／使用者事件、自動化流程、權限、用量、停用、開發者文件與測試環境的後端側。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：bot_applications、bot_webhooks、bot_events。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：Official Account、Message、Chat、User。
// 禁止事項：不得以 Bot Token 冒充使用者身份；不得繞過訊息權限檢查發訊；不得將 Bot 事件明文（E2EE 內容）寫入 bot_events。
// 所屬階段：Phase 08（社群貼圖商業）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package bot
