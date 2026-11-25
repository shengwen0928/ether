// Package notification 是 Notification（通知） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：通知的產生、偏好、推播 Token 管理、投遞紀錄、App 內通知、分類、已讀／清除、靜音時段、個別聊天室控制。推播內容須尊重隱私設定，關閉預覽時不得洩漏內容。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：notifications、notification_preferences、push_tokens、notification_deliveries。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User、Device Management、Profile。
// 禁止事項：不得直接依賴 Message、Group、Call 等業務模組（素材一律經事件通用封包）；不得在推播內容中放入 E2EE 明文。
// 所屬階段：Phase 05–06（推播扇出，經事件訂閱落地）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package notification
