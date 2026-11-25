// Package moderation 是 Moderation（內容審核） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：檢舉受理、審核處分（帳號限制、暫時／永久停權、內容處置）、垃圾訊息偵測與惡意連結警告的規則側、申訴流程入口。處分狀態的唯讀查詢供 Message 等模組在寫入前檢查。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：reports、moderation_actions。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User（直接依賴僅此；其他模組素材經事件訂閱）。
// 禁止事項：不得讀取 E2EE 訊息明文；不得以審核為由繞過權限讀取私人對話；不得替 Administration 做管理員權限裁決。
// 所屬階段：Phase 08（檢舉入口）–Phase 10（管理處置）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package moderation
