# CI 草稿說明（infrastructure/ci-drafts/）

## **強制流程：這些草稿絕對不得以 push 方式建立 `.github/workflows/`**

**GitHub App 無 `workflows` 權限，以 push 方式建立或修改 `.github/workflows/`
下的檔案會回 403（見 `~/AGENTS.md` 既有教訓）。因此：**

- **本目錄的草稿只能留在本目錄，不得複製、移動或以任何 push 方式寫入
  `.github/workflows/`。**
- **正式啟用時，必須經雲端瀏覽器＋使用者登入工作階段，在 GitHub 網頁上手動
  建立對應的 workflow 檔案**（以本目錄草稿內容為底稿，逐字核對後貼上）。
- **絕對不要建立 `.github/` 目錄。** 本專案目前沒有 `.github/` 目錄；
  `make verify-static` 會檢查 `.github` 不存在，作為防呆。

## 草稿未經 CI 實跑驗證

兩份草稿皆未在 GitHub Actions 實跑過。Action 版本標籤、Go 版本
（`1.25.0`，須與 Dockerfile 的 `golang:1.25.0-alpine3.21` 對齊）皆於
2026-11-25 固定為草案，首次實跑時須覆核是否仍受支援；升級須同步改
Dockerfile 與草稿，不得單改一處。程式碼尚無、compose 對齊缺口
（`docs/phase01/compose-alignment-notes.md` G-01／G-02）未解決前，建置步驟
預期失敗，不得以紅燈為常態合併。

## 草稿內容

| 草稿 | 觸發 | 內容 |
|---|---|---|
| `github-actions-build.yml` | push／PR 至 `main`、手動觸發 | Go 建置／`gofmt`／`go vet`／`go test`；Docker 映像建置（不推送，三服務）；`docker compose config` 驗證；Secrets 掃描 |
| `github-actions-pr-checks.yml` | PR 至 `main`、手動觸發 | PR 閘門：Go 檢查＋單元測試、`make verify-static`、Compose config 驗證、`.env` 未被追蹤檢查；Secrets 掃描；依賴掃描（示例） |

## 定稿前待決定事項

1. 分支策略：草稿以 `main` 為目標分支；儲存庫尚未建立，分支名與保護規則
   待定稿（CI 失敗不得合併的合併規則亦待 Phase 01 定稿，見
   `docs/testing/TEST_STRATEGY.md` §7）。
2. Secrets 掃描與依賴掃描工具待 Phase 11 定稿（見 TEST_STRATEGY.md §9）；
   草稿中的 gitleaks／govulncheck 僅為示例，不得視為已定。
3. Compose 對齊裁決（G-01）：`docker/build-push-action` 的 context 目前按
   獨立 Dockerfile 路徑書寫，與現行 compose 參照不一致；裁決後須同步修正
   草稿或 compose（compose 的修改須經批准，Infra 軌不得擅自改）。
4. 映像推送：草稿只建置、不推送；映像倉庫選定待定稿（見
   `docs/deployment/CLOUD_ARCHITECTURE.md` §9），不得預寫倉庫位址。
5. iOS 建置走 macOS runner（見 ADR-003），不在本目錄草稿範圍，待 App 軌／
   CI 定稿時另立草稿，不得以 Linux runner 假裝可建 iOS。
