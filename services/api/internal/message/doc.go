// Package message 是 Message（訊息）模組的 Phase 05 實作。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.7）：訊息生命週期：送出、狀態機
// （伺服器端僅 sent／recalled；Pending／Sending 等為客戶端狀態，
// 不入庫）、編輯、收回、刪除（僅自己檢視）、引用／回覆、已讀／
// 送達回條、Sequence 分配、冪等去重（client_message_id）、表情
// 反應、置頂、附件關聯、轉傳。附件關聯須經 Media 模組驗證檔案
// 可關聯性（掃描閘門 fail-closed）；轉傳不複製附件。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：messages、
// message_receipts、message_replies、message_reactions、
// message_pins、message_attachments、message_forwards；另有 message_outbox（Outbox
// 事件表，DATA_FLOW.md §10 缺口 1 待 ERD 定案，本模組擁有，
// 見 migrations/000009 檔頭與 internal/outbox）。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：
// Chat（容器與成員驗證）、User、Media（附件詮釋資料，後續里程碑）、
// Friend（封鎖檢查）、Moderation（唯讀處分查詢，後續里程碑）。
// 禁止事項：不得直接讀寫 conversations 表（最後訊息快取經 Chat
// 介面更新）；不得以 WS 連線存在代替送達保證；不得物理刪除訊息列
// （append-only）；不得在 E2EE 聊天室將明文寫入伺服器欄位或日誌。
//
// E2EE 明確邊界：本里程碑不實作任何密碼學功能。content 一律視為
// 不透明字串傳輸與儲存，不得撰寫、拼裝或宣稱 Signal Protocol
// 實作；金鑰分發與棘輪狀態不在範圍，encryption_mode 待 E2EE 文件
// 定稿（見 E2EE_RESEARCH.md）。不得以本模組的存在宣稱訊息已加密。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// 推播中繼：寫入路徑以同一交易寫入 message_outbox 事件（見 events.go
// 與 internal/outbox），發佈迴圈見 internal/relay（Publisher 介面，
// NATS 發佈者待落地）；Gateway 訂閱成員驗證見 realtime-gateway 的
// internal/hub（MembershipChecker 介面，api 側查詢待整合）。
// 實作見 message.go（領域＋記憶體儲存）、postgres.go（PostgreSQL
// 儲存庫）、handler.go（HTTP 端點）；儲存語義與 ERD 差異見
// migrations/000006_messaging.up.sql 檔頭。
package message
