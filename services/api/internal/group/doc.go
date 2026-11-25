// Package group 是 Group（群組）模組的 Phase 06 實作（後端離線可驗證部分）。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.8）：群組建立與管理、角色
// （Owner、Admin、Member）與權限、邀請、公告、成員管理、離開／
// 解散、所有權轉移。群組聊天經 Chat 介面協作（建立時建對話容器、
// 成員異動時同步對話成員），不得直接寫入 conversations／messages
// 表；不得以群組角色代替對話成員權限；不得替 Community 模組做決策。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：groups、group_members、
// group_invitations、group_announcements、group_settings
// （group_roles 屬自訂角色後續里程碑，本里程碑不建表，角色為固定
// 列舉，見 migrations/000007 檔頭）。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：
// User、Chat、Friend（皆為唯讀／協作介面，不得直連他模組資料表）。
// 禁止事項：不得直接寫入訊息表；不得以群組角色代替對話成員權限；
// 投票與社群（Community）不在本里程碑範圍，不得實作。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// 實作見 group.go（領域＋記憶體儲存）、postgres.go（PostgreSQL
// 儲存庫）、handler.go（HTTP 端點）；儲存語義與 ERD 差異見
// migrations/000007_groups.up.sql 檔頭。
package group
