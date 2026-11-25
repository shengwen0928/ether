// Package contact 是 Contact（聯絡人） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：聯絡人同步（須明確授權、限制範圍）、同步紀錄、基於同步結果的推薦輸入（雜湊比對，不得上傳明文號碼）。同步結果只作推薦輸入，好友建立仍走 Friend 邀請流程。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：contact_sync_records。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User、Profile、Friend。
// 禁止事項：不得儲存明文電話號碼；未經授權不得使用通訊錄資料；拒絕通訊錄權限不得阻擋其他功能；不得自動加好友。
// 所屬階段：Phase 04（好友聯絡人）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package contact
