// Package timeline 是 Timeline（動態，ETHER Moments）模組的實作
// （Phase 08 社群貼圖商業，離線可驗證部分）。
//
// 擁有資料表：posts、post_media、post_likes、post_comments
// （migration 000012；post_shares／post_bookmarks 屬後續里程碑）。
// 允許依賴（唯讀介面）：User、Friend、Media。禁止事項：不得向被
// 封鎖或無權限者展示貼文；留言不得寫入 messages；不得替
// Moderation 做處分裁決。動態無對應 WebSocket 事件名，不寫
// Outbox 事件。
package timeline
