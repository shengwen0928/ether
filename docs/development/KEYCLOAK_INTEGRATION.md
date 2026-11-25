# Keycloak 整合（M2b-3）

狀態：程式接線已完成（M2b），實機整合**尚未驗證**（需隔離環境的
Keycloak 實例）。本文件是隔離環境啟動與驗證的依據，不得以 Mock
測試通過冒充實機通過。

## 1. 設定一覽

| 設定 | 環境變數 | 值／規則 |
|---|---|---|
| Keycloak 位址 | `KEYCLOAK_URL`（compose 組裝） | 例如 `http://keycloak:8080`（容器內） |
| Realm | `KEYCLOAK_REALM` | `ether`（暫定預設；隔離方式待定稿，見 IDENTITY §12 缺口 11） |
| Issuer | 由位址＋Realm 組裝 | `<KEYCLOAK_URL>/realms/<KEYCLOAK_REALM>`；須與 discovery 回傳的 `issuer` 完全一致，否則 `buildVerifier` 拒絕啟動 |
| API Client 識別碼 | `KEYCLOAK_CLIENT_ID` | `ether-api`（草案；RS256 驗證不需此值，供文件與範本一致性使用） |
| Audience 強制值 | `KEYCLOAK_AUDIENCE` | 正式部署設為 `ether-api`；留空＝不強制 aud（M1 預設行為，僅開發用） |
| Web Client | Realm 範本 | `ether-web`（公開 Client、授權碼＋PKCE S256） |
| 身分儲存後端 | `IDENTITY_STORE` | `postgres`（預設）／`memory`（開發用，資料不持久） |

設定與程式碼分離：全部設定皆經環境變數（`internal/config`），程式碼
內不得寫入 Realm 名、Client 識別碼、Redirect URI 或任何密鑰；測試用
密鑰與預設密碼不得寫入正式設定（`.env.example` 全部為佔位符）。

## 2. Realm 範本

`infrastructure/docker/keycloak/realm-ether.template.json` 為 Realm
匯入範本（僅供隔離／開發環境）：

- Realm `ether`；`ether-web` 公開 Client（PKCE S256、Redirect URI 僅
  `http://localhost:5173/*`）；`ether-api` 為 audience 識別用 Client
  （不開啟任何流程、不含密鑰）。
- **範本限制**：Redirect URI 與 Web Origins 皆為本機開發佔位符，正式
  部署須改為正式網域；不得將範本內的佔位符當作正式設定。
- 匯入後須在管理主控台核對：Realm 名、兩個 Client 的識別碼與流程
  開關、PKCE 方法為 S256。

## 3. 流程對照

| 流程 | Keycloak 側 | ETHER API 側 |
|---|---|---|
| 登入 | 簽發 RS256 Access／Refresh Token（含 `sub`、`email`、`email_verified`、`sid`） | `RequireAuth` 驗 JWKS 簽名、iss、exp、aud（有設定時）；`Provision` 依 `sub` 建置帳號（JIT） |
| Session 映象 | `sid` 為 Keycloak 工作階段識別碼 | `user_sessions` 存雜湊映象；`POST /api/v1/auth/logout` 須 Token 含 `sid`，只撤銷 ETHER 映象（不代 Keycloak 執行 back-channel 登出，該機制待定稿） |
| Refresh 輪換 | 簽發新 Token 對 | `SessionService.Rotate`：舊雜湊標記已消耗（`session_refresh_hashes`，持久化）、寫入新雜湊；重放舊雜湊 → 家族撤銷 |
| 撤銷 | Keycloak 撤銷 Token | ETHER 映象標記撤銷；**請求期強制檢查 ETHER 撤銷狀態為 M2 待辦**（M1 中介層只驗 Token，見 STATUS.md），不得宣稱已即時同步撤銷 |

## 4. 隔離環境驗證清單（實機）

依序執行，全部通過才算「實機驗證通過」：

1. 啟動 Keycloak（compose `keycloak` 服務）與 PostgreSQL；匯入 Realm 範本。
2. `cmd/migrate status` 顯示 4 個版本皆已套用（`migrate up` 後）。
3. 以 `ether-web` 完成一次真實登入（授權碼＋PKCE），取得 Access Token。
4. Token 的 `iss` 等於 `<KEYCLOAK_URL>/realms/ether`；`aud` 含 `ether-api`
   （若無，須在 Keycloak 為 `ether-web` 加上 audience mapper 後重測，
   不得改程式繞過）。
5. `go test -tags integration ./internal/auth/ -run 'TestLiveKeycloak'`（設定
   `KEYCLOAK_TEST_ISSUER` 與 `KEYCLOAK_TEST_TOKEN`）通過。
6. 以該 Token 呼叫 `GET /api/v1/auth/me`（`IDENTITY_STORE=postgres`）：
   帳號建置成功、再次呼叫不重複建帳號；`GET /api/v1/auth/sessions`
   與裝置端點回應正確。
7. 重新整理 Token 後，舊 Refresh 雜湊重放應觸發家族撤銷（以
   `GET /api/v1/auth/sessions` 的 `is_revoked` 核對）。
8. `IDENTITY_STORE=memory` 啟動時日誌明示記憶體模式；正式部署不得
   使用 memory。

## 5. 測試區分

- Mock 測試（`internal/auth/oidc_test.go`，無 tag）：以本機 RSA 金鑰
  與 httptest JWKS 伺服器驗證邏輯（簽名、iss、exp、aud 字串／陣列、
  kid 輪換、alg 混淆）。通過 ≠ 實機通過。
- 實機測試（`internal/auth/oidc_integration_test.go`，`integration`
  tag）：連接 `KEYCLOAK_TEST_ISSUER` 指定的 Keycloak，驗 discovery、
  JWKS 與真實 Token。無環境變數時跳過並標示「環境待驗證」。
- 資料庫整合測試（`integration` tag，`ETHER_TEST_DATABASE_URL`）：
  驗 migration 套用／回滾與儲存庫持久化，同樣需隔離資料庫。
