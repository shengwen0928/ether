// Package profile 是 Profile（個人檔案） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：公開檔案（名稱、唯一 ID、頭像、背景、簡介、狀態訊息、QR Code、分享連結）、可見範圍控制、生日的可選公開範圍。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：user_profiles。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User。
// 禁止事項：不得存放電話、Email 等私密聯絡資料；不得替 Friend 模組判斷好友關係；不得在公開檔案回應中洩漏私密欄位。
// 所屬階段：Phase 03（身分帳號）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package profile
