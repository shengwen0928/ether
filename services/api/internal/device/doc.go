// Package device 是 Device Management（裝置管理） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：登入裝置清單、裝置信任、遠端登出（裝置側）、多裝置同步的裝置註冊側、裝置撤銷。裝置是 Session 與推播 Token 的錨點之一。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：user_devices。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User。
// 禁止事項：不得管理 Session 本身（屬 Authentication）；不得替 Notification 管理推播 Token；不得在裝置紀錄中存放 Token 明文。
// 所屬階段：Phase 03（身分帳號）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package device
