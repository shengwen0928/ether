狀態：草案／待批准

# ADR-010：Keycloak 身分架構

- **狀態**：已決定（草案／待批准）
- **日期**：2026-11-25 00:33（使用者架構調整指令）
- **依據**：SPEC-SOURCE.md 頂部「架構調整」一節、§05（帳號系統）、身分與權限模型文件（另冊）

## 問題

身分驗證應如何改採 Keycloak，同時保留既有的帳號能力且不破壞使用者／好友／裝置關聯？

## 選項

| 選項 | 說明 |
|---|---|
| A. Keycloak 承接身分落實，業務關聯不變 | 身分服務與業務主檔分離，關聯語義保留 |
| B. 自寫一套身分系統取代 Keycloak | 與架構調整指令衝突；且自寫身分系統的安全風險高 |
| C. Keycloak 另建一套使用者主檔，與 `users` 並存 | 破壞關聯：好友／裝置／訊息皆參照 `users`，兩套主檔必生不一致 |

## 決定

選 A：身分驗證改 Keycloak。保留 Email 驗證、登入登出、OAuth（Google／Apple）、MFA、Session、Access／Refresh Token、裝置管理、遠端登出、密碼重設；不得破壞使用者／好友／裝置關聯。

## 架構銜接

- **職責切分**：Keycloak 負責身分落實（驗證、Token 核發、MFA、OAuth 整合、Session 概念）；Authentication 模組經 Keycloak 介面對外提供統一的 Token 驗證與授權判斷介面（供 API／Gateway），不得自寫一套身分儲存與 Keycloak 並存（見 `MODULE_BOUNDARIES.md` §5.1）。
- **關聯不變**：`users` 仍為全系統身分根；`friendships`／`friend_requests`／`blocked_users`／`hidden_users`、`user_devices`、`user_sessions` 的參照語義不變；Keycloak 的使用者識別須穩定對應至 `users`（對應方式待身分文件定案，不得以 Email 作唯一對應鍵——Email 可變更，細節待定）。
- **管理員分離**：管理後台認證仍獨立（`admin_users` 體系），不得與 Keycloak 的一般使用者 Realm 混用；管理員 Realm／Client 的隔離方式待身分文件定案。
- **Token 與 Session**：短效 Access Token＋Refresh Token Rotation 保留；Refresh Token 重放偵測與全裝置撤銷語義保留（見 `DATA_FLOW.md` §2）；Keycloak 事件與 `user_sessions`／`user_devices` 寫入的時序待定（見 `DATA_FLOW.md` §11 缺口 8）。
- **部署**：Keycloak 以容器部署於同一 Docker Compose＋Coolify 架構內；Realm／Client 設定為部署設定（版本控管，無 Secrets）；Secrets 經 OpenBao 注入（見 `SECURITY_ARCHITECTURE.md` §6）。

## 理由

1. 使用者指令明定 Keycloak 承接身分，且明文要求保留既有能力與關聯——本決定只是落實該指令的邊界。
2. 身分與業務分離是縱深防禦的一部分：身分服務被攻破不等於業務資料可任意讀寫（業務權限仍由各模組裁決）。

## 後果

- 身分文件的 Token 格式、Session 同步、裝置撤銷細節，須以本 ADR 的銜接邊界為準，不得另起一套。
- Keycloak 的版本固定、備份（Realm 設定與資料）、升級流程待部署文件定案；驗證為「待 CI 或有 Docker 的環境驗證」。
- 本決定不涉及花錢或綁卡；Keycloak 自架的主機成本屬 ADR-009 的伺服器成本，待查證。
