// Package sticker 是 Sticker（貼圖）模組的 Phase 08 實作：貼圖包、
// 貼圖、使用者收藏、最近使用、分類、搜尋。購買交易本身屬 Payment
// Integration（擁有 sticker_purchases 表），本模組不得處理付款；
// 授權發放只經 GrantPack／RevokeGrant 介面（供 Payment 模組付款
// 完成後呼叫），不得在未授權情況下發放付費貼圖。
//
// 責任邊界（MODULE_BOUNDARIES.md §5.11）：貼圖包、貼圖、使用者
// 收藏（user_stickers）、最近使用、分類、搜尋。商店展示排序與
// 推薦、排行榜、創作者頁、分享連結、下載計數商業統計皆屬後續
// 里程碑，本里程碑不實作。內容一律原創，不得含第三方受保護素材
// （本模組只存 media_file_id 參照，不審查素材來源，素材合規屬
// 內容治理流程）。
//
// 擁有資料表（ERD §6.7 草案，差異見 migrations/000014 檔頭）：
// sticker_packs、stickers、user_stickers。
// 允許依賴（依賴 DAG，只允許箭頭方向）：User、Media，皆只經唯讀
// 介面，不得直讀他模組資料表。
// 禁止事項：不得處理付款；不得在未授權情況下發放付費貼圖；不得
// 儲存貼圖位元組（只存 media_file_id）；不得寫入 Outbox 事件
// （§21 的 15 個事件皆為訊息事件，無貼圖事件名）。
package sticker
