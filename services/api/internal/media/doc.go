// Package media 是 Media（媒體）模組的 Phase 06 實作（後端離線可驗證部分）。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.10）：媒體檔案的詮釋資料、
// 上傳會話、縮圖、存取紀錄、權限驗證的檔案側。實體檔案存於
// MinIO 物件儲存，資料表只存詮釋資料與路徑；物件鍵的核發與結構
// 由本模組定，他模組不得自行拼物件鍵或直連 MinIO。
// 擁有資料表（SPEC-SOURCE.md §20，表名為準）：media_files、
// media_uploads、media_thumbnails、media_access_logs。
// 允許依賴（依賴 DAG，MODULE_BOUNDARIES.md §2，只允許箭頭方向）：
// User（唯讀存在確認，不得直連他模組資料表）。
// 禁止事項：不得判斷訊息／貼文／群組的業務權限（只提供檔案側
// 檢查，業務權限由呼叫方裁決）；不得在未掃描完成前開放檔案分發；
// 不得儲存信用卡等付款資料。
// 模組間呼叫只經公開介面；反向通知一律走 NATS 事件，不得直接呼叫回頭。
// 實作見 media.go（領域＋記憶體儲存）、objectstore.go（物件儲存
// 介面＋記憶體 stub）、minio.go（MinIO／S3 相容轉接器，標準庫
// SigV4 簽名）、postgres.go（PostgreSQL 儲存庫）、handler.go
// （HTTP 端點）；儲存語義與 ERD 差異見 migrations/000008_media.up.sql
// 檔頭。位元組通道為 presigned 直傳（§2z）：MinIO 模式核發簽名 URL，
// 記憶體模式以 api 內中繼端點承載位元組。
// 掃描實作與縮圖產生不在本里程碑範圍，僅留狀態欄位與介面。
package media
