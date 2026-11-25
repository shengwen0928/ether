// Package call 是 Call（通話）模組的實作（Phase 07 離線可驗證部分）。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.12）：通話會話管理、通話紀錄、
// 參與者、通話事件、信令協調（經 Gateway）、來電／接聽／拒接／
// 忙線／未接狀態、通話品質資料的接收。媒體面屬 LiveKit，本模組只
// 負責會話與紀錄。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：calls、
// call_participants、call_events。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：
// User、Chat、Group、Authentication、Device Management。
// 禁止事項：不得儲存或記錄通話內容；不得替 LiveKit 做媒體轉發決策；
// 不得在信令中夾帶 Token 或金鑰材料；不得繞過 Friend 封鎖檢查發起
// 通話（封鎖檢查經 Chat 介面）。
// 所屬階段：Phase 07（語音視訊會議）。本里程碑實作會話與紀錄範圍
// （發起／接聽／拒接／忙線／未接／結束、參與者、事件、紀錄查詢）；
// LiveKit 房間建立、Access Token 簽發、媒體品質資料寫入、即時信令
// 皆未實作（信令介面只留 SignalCoordinator 佔位）。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
package call
