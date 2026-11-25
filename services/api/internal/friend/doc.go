// Package friend 是 Friend（好友）模組的 Phase 04 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.4）：好友邀請（發送／接受／拒絕／
// 取消／過期／紀錄）、好友關係管理、封鎖、隱藏、備註、分類、好友
// 搜尋與排序。封鎖／刪除／隱藏是三種不同狀態，不得混為一談。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：friendships、
// friend_requests、blocked_users、hidden_users。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：
// User、Profile、Authentication。
// 禁止事項：不得以 Friend 關係代替 Chat 的成員權限判斷；不得向
// 被封鎖者揭露封鎖狀態；不得把聯絡人同步資料寫入好友表。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// 實作見 friend.go（領域＋記憶體儲存）、postgres.go（PostgreSQL
// 儲存庫）、handler.go（HTTP 端點）；儲存語義與 ERD 差異見
// migrations/000005_friends.up.sql 檔頭。
package friend
