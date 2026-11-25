// Package auth 是 Authentication（認證） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：註冊／登入驗證、密碼／Email／手機 OTP 驗證、OAuth（Google／Apple）、MFA、Refresh Token Rotation、Session 管理、登入紀錄、改密／重設、停用／刪除的認證側；提供全系統唯一的身分與授權介面。身分能力由 Keycloak 承接，不得自寫一套身分儲存與 Keycloak 並存。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：user_sessions、user_security_keys。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User、Device Management。
// 禁止事項：不得直接讀寫 users 以外的使用者業務欄位；不得替他模組判斷業務權限；不得儲存明文密碼；不得將 Session 或 Token 寫入日誌。
// 所屬階段：Phase 03（身分帳號）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package auth
