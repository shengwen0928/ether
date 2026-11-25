// Package officialaccount 是 Official Account（官方帳號） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：官方帳號註冊、名稱／頭像／認證標章／介紹、搜尋、加入、封鎖、檢舉入口、客服會話的帳號側。官方訊息經 Message 送出，不得自建訊息表。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：official_accounts、official_account_members。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User、Profile、Chat、Message。
// 禁止事項：不得自建訊息或通知表；不得以官方帳號身份繞過封鎖或隱私設定；不得替 Bot 模組管理 Token 與 Webhook。
// 所屬階段：Phase 08（社群貼圖商業）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package officialaccount
