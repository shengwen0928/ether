// Package user 是 User（使用者） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：使用者主檔（帳號本體）的建立、帳號狀態、帳號層設定、資料匯出與帳號刪除流程的帳號側協調。全系統的身分根。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：users、user_settings。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：無（僅依賴基礎設施）。
// 禁止事項：不得依賴 Profile、Friend 或任何上層模組；不得在 users 內存放公開檔案、好友關係或裝置資料；不得以 users 兼作管理員表。
// 所屬階段：Phase 03（身分帳號）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package user
