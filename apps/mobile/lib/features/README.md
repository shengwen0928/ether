# Features — Phase 01 佔位對照

狀態：全部「Phase 01：空骨架」。各 feature 目錄內不得放功能程式碼。

## 目錄與負責階段

目錄劃分依 `SPEC-SOURCE.md` §26；負責階段依 `docs/ROADMAP.md`（手冊 B–N）。
「負責階段」為該功能落地驗收的階段，Phase 01 只保留空目錄。

| 目錄 | 對應規格 | 負責階段 | 狀態 |
|---|---|---|---|
| `auth/` | §05 帳號系統 | Phase 03 身分帳號（手冊 E） | Phase 01：空骨架 |
| `chat/` | §07 一對一聊天 | Phase 05 即時訊息核心（手冊 G） | Phase 01：空骨架 |
| `contacts/` | §06 好友系統（聯絡人同步） | Phase 04 好友聯絡人（手冊 F） | Phase 01：空骨架 |
| `friends/` | §06 好友系統 | Phase 04 好友聯絡人（手冊 F） | Phase 01：空骨架 |
| `groups/` | §08 群組 | Phase 06 媒體群組社群（手冊 H） | Phase 01：空骨架 |
| `communities/` | §09 社群 | Phase 06 媒體群組社群（手冊 H） | Phase 01：空骨架 |
| `calls/` | §10 通話 | Phase 07 語音視訊會議（手冊 I） | Phase 01：空骨架 |
| `timeline/` | §12 動態 | Phase 08 社群貼圖商業（手冊 J） | Phase 01：空骨架 |
| `stickers/` | §11 貼圖與主題 | Phase 08 社群貼圖商業（手冊 J） | Phase 01：空骨架 |
| `notifications/` | §14 通知 | 未單列階段；隨 Phase 05 即時訊息與 Phase 06 能力落地，確切劃分待 Phase 00 審查確認 | Phase 01：空骨架 |
| `settings/` | §18 設定中心 | 未單列階段；設定中心能力隨各功能階段落地 | Phase 01：空骨架 |
| `navigation/` | §04 導覽架構（底部五入口） | 骨架路由在 `lib/main.dart`（單一佔位路由）；完整五入口導覽待 Phase 02 設計系統與後續階段 | Phase 01：空骨架 |

## 規則

- 不得在 feature 目錄內寫功能程式碼、假資料、Mock 後端或無作用按鈕。
- 不得建立聊天／好友等功能頁；路由目前只有 `lib/main.dart` 的一個骨架佔位頁。
- 各 feature 的資料模型、API、狀態管理、錯誤處理、測試，皆待其負責階段實作與驗收。
- 封鎖／刪除／隱藏是不同狀態，不得混為一談（§06）；此語義待 Phase 04 建模，不得在本階段預寫。
