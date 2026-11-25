// Package admin 是 Administration（管理） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：管理員帳號、角色與權限、操作稽核、管理後台的認證與授權。管理操作本身經各業務模組的公開介面執行，本模組只負責「誰有權做」與「做了什麼的稽核」，不重寫業務邏輯。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：admin_users、admin_roles、admin_permissions、admin_audit_logs。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：無直接業務模組依賴。
// 禁止事項：不得任意讀取 E2EE 訊息；不得與一般使用者共用 Token 或 Session；不得在前端判斷管理員權限；不得刪除或竄改 admin_audit_logs。
// 所屬階段：Phase 10–11（管理後台，見 ROADMAP.md §3.11）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package admin
