# ETHER Secrets 管理（OpenBao 接入骨架，Phase 01）

狀態：慣例與文件，待 CI 或有 Docker 的環境驗證。本文件只寫慣例，不得寫入
任何真實 token、金鑰、密碼、解封金鑰。全部值以佔位符（`change-me`／
`example.invalid`）表示；正式值一律以 OpenBao／環境注入提供，不得寫入 Git、
不得寫入映像、不得寫入日誌。

依據：`docs/deployment/CLOUD_ARCHITECTURE.md` §6–§7（Secrets 管理與環境劃分）、
`docs/deployment/SELF_HOSTING.md` §2／§4、`SPEC-SOURCE.md` 手冊 C（環境變數／
Secrets 管理）、`.env.example`（範本與佔位符慣例）。

## 1. KV 路徑慣例

OpenBao KV（版本化 KV，mount 名 `secret`，見 `.env.example` Phase 01 附加段落
的 `OPENBAO_KV_MOUNT` 佔位符）路徑慣例如下。路徑本身不含任何真實值；
`<env>` 取值只限 `local`／`development`／`staging`／`production` 四者
（環境劃分見 CLOUD_ARCHITECTURE.md §6），不得以程式碼分支區分環境。

```text
secret/ether/<env>/postgres        # 資料庫憑證（使用者、密碼、資料庫名）
secret/ether/<env>/valkey          # Valkey 憑證（如啟用認證，方式待定稿）
secret/ether/<env>/nats            # NATS 憑證（如啟用認證，方式待定稿）
secret/ether/<env>/minio           # MinIO 憑證（root／應用存取金鑰）
secret/ether/<env>/meilisearch     # Meilisearch 金鑰
secret/ether/<env>/keycloak        # Keycloak 管理憑證、Client Secret
secret/ether/<env>/signing         # JWT／Token 簽章金鑰
secret/ether/<env>/push            # 推播憑證參照（FCM／APNs，檔案或參照，不得寫值）
secret/ether/<env>/external-ai     # 外部 AI API 金鑰（可選擴充，僅使用者決定啟用時）
secret/ether/<env>/turn            # TURN 憑證（待 Phase 07 定稿）
secret/ether/<env>/backup          # 備份加密金鑰
```

- 每條路徑的欄位名（key）與輪換週期待定稿，定稿時逐項確認存放位置與
  輪換週期，不得在本文件預寫欄位名為已定。
- 備份加密金鑰與解封金鑰不得同處存放；解封金鑰不得寫入 Git（見
  CLOUD_ARCHITECTURE.md §8.1）。
- E2EE 金鑰不在本慣例範圍內：E2EE 金鑰管理見 E2EE 研究文件，不得以
  OpenBao 慣例假裝已覆蓋 E2EE 金鑰生命週期。

## 2. Policy 命名慣例

服務以最小權限讀取所需 Secrets，不得讀取無關 Secrets；讀取須有稽核紀錄
（見 CLOUD_ARCHITECTURE.md §7）。

```text
ether-<env>-api        # api：postgres、valkey、nats、minio、meilisearch、keycloak、signing
ether-<env>-gateway    # realtime-gateway：postgres、valkey、nats、signing（讀取範圍待定稿）
ether-<env>-worker     # worker：postgres、valkey、nats、minio、meilisearch
ether-<env>-readonly   # 唯讀稽核／維運：僅讀取，不寫入（範圍待定稿）
```

- Policy 名中的 `<env>` 與 KV 路徑的 `<env>` 取值一致；`local` 環境不得讀取
  `staging`／`production` 的路徑。
- Policy 的具體 capabilities（read／list）與認證方式（Token／AppRole 等）
  待定稿，不得在本文件預寫為已定；認證方式定稿前，不得以開發模式 Token
  假裝正式認證已完成。

## 3. 開發／staging／production 的注入方式

| 環境 | 注入方式 | 說明 |
|---|---|---|
| local | 本機 `.env`（佔位符，見 `.env.example`）＋本機 OpenBao（或開發模式） | 不得以正式 Secrets 在本機開發；`.env` 不進 Git |
| development | OpenBao 開發 Vault，不得入 Git | 共享開發驗證用虛構測試資料 |
| staging | OpenBao，不入 Git | 與 production 相同的部署流程，成本待查證 |
| production | OpenBao，不入 Git；Production Secrets 絕對不入 Git | 正式 Secrets 的存取權限最小化，不得共用管理帳號 |

- Compose 檔案只寫參照（`${VAR}`），不寫值；各環境的實際值以 OpenBao／
  環境注入提供，不得寫入映像（見 CLOUD_ARCHITECTURE.md §6）。
- `.env.example` 上半部既有的 `OPENBAO_ADDR`／`OPENBAO_TOKEN` 註解行保留為
  位址與 Token 的慣例位置；Phase 01 附加段落不再重複定義，以免兩處不一致。
  正式環境的位址與 Token 一律以環境注入提供，不得寫入 `.env.example`。

## 4. 與 Go 端 Secrets Provider 介面的對應

以下對照程式碼軌已建立的骨架 `services/api/internal/secrets/secrets.go`
（本軌未撰寫、未驗證編譯；以下為靜態對照，介面變更時以程式碼為準）：

- `Provider` 介面（已建立）：`Get(ctx context.Context, name string) (string, error)`。
  `name` 為 secret 的鍵名（例如環境變數名一類，實際鍵名以部署定稿為準）；
  找不到或未設定時回傳 error，不得回傳空字串當作值，錯誤中不得包含
  secret 值。
- `EnvProvider`（已建立，本機開發用）：自環境變數讀取；正式環境須改用
  OpenBao Provider，不得以環境變數承載正式 secret 而不經部署定稿。
- `OpenBaoProvider`（已建立，stub）：`NewOpenBaoProviderFromEnv` 讀取
  `OPENBAO_ADDR` 與 `OPENBAO_TOKEN`（`.env.example` 上半部註解中的名稱）；
  `Get` 現階段永遠回傳錯誤（`ErrNotConfigured` 或「client not implemented」），
  絕不回傳 secret 值，不得假裝已接線。正式接線（客戶端、解封流程、Token
  注入）待部署定稿。
- **對齊缺口（待定稿）**：本文件 §1 的 KV 路徑慣例（`secret/ether/<env>/…`）
  與程式碼 `Get` 的「鍵名」參數尚未對應——路徑如何對映為鍵名、一次讀取
  取整條路徑抑或單一欄位，待部署定稿，不得預寫對映規則。
- 啟動期行為（待定稿）：Secrets 讀取失敗時，服務不得以預設假值啟動、不得
  以空值繼續提供服務；失敗處理（重試／退出）待後階段定稿，不得預寫。
- 快取與輪換（待定稿）：Provider 是否快取、快取期限、輪換時雙金鑰過渡期
  規則，皆待定稿；不得以「讀一次即永久有效」假裝輪換已處理。

## 5. Compose 中 OpenBao 開發模式佔位的限制

`docker-compose.yml` 的 `openbao` 服務以 `command: ["server", "-dev"]` 佔位：

- 僅供本機開發佔位；正式環境的 OpenBao 伺服器模式、解封流程待定稿，
  **不得以開發模式部署正式環境**（compose 檔內註解與 SELF_HOSTING.md §9
  第 4 項已有相同限制，本文件重申，不放寬）。
- 開發模式的資料為記憶體／臨時性質（volume 掛載見 compose 對照筆記），
  不得假裝開發模式的 Secrets 具備正式環境的持久性與備份能力。
- 正式模式定稿前，`openbao` 服務的 healthcheck（`/v1/sys/health`）僅證明
  開發模式行程存活，不證明正式解封／備份流程可用。

## 6. 缺口（只列、不補決定）

1. KV 各路徑的欄位名、輪換週期、雙金鑰過渡期規則，待定稿。
2. Policy 的 capabilities 與認證方式（Token／AppRole 等），待定稿。
3. 正式模式 OpenBao 的伺服器模式、解封流程、主金鑰保管、備份與還原，待定稿。
4. Go 端 Provider 介面已建立為 stub（見 §4 對照），正式接線、啟動期失敗處理、
   快取與輪換、KV 路徑與鍵名的對映，待程式碼軌／部署定稿。
5. 本文件全部慣例待有 Docker 的環境驗證（OpenBao 開發模式連通、policy
   唯讀／唯寫邊界），不得宣稱已驗證。
