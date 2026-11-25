// Package community 是 Community（社群） 模組的 Phase 01 佔位文件。
//
// 責任邊界（MODULE_BOUNDARIES.md §5）：社群建立與管理、公開／私密、分類、搜尋、邀請／審核、成員與分級權限、多聊天室（頻道）、子群組、討論區、活動、投票、檢舉入口、黑名單、資料分析入口、操作紀錄。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：communities、community_members、community_channels、community_roles、community_invitations。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：User、Chat、Profile。
// 禁止事項：不得依賴 Group 模組（頻道不得複用群組表）；不得直接寫入訊息表；不得以社群角色代替全平台管理員權限。
// 所屬階段：Phase 06（媒體群組社群）；本階段（Phase 01）只放本 doc.go，不得寫功能程式碼，
// 不得直接讀寫他模組的資料表，不得 import 他模組的內部套件。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// TODO：功能實作、公開介面、事件訂閱登記，皆待所屬階段落地。
package community
