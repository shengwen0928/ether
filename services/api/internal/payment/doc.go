// Package payment 是 Payment Integration（付款整合） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：付款供應商 Adapter（合規第三方，不得自行儲存信用卡完整資訊）、交易紀錄、訂單通知的觸發、付款狀態查詢。生活服務的付款入口預留在本模組，不得超前實作未定的商業流程。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：sticker_purchases。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User、Sticker。
// 禁止事項：不得自行儲存信用卡完整資訊；不得直接依賴特定付款供應商 SDK（須經 Adapter）；不得在付款未確認前發放授權。
// 所屬階段：Phase 08（社群貼圖商業）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package payment
