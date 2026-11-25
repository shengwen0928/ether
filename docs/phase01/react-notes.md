# Phase 01 React 軌道筆記（react-notes.md）

軌道：React（`apps/web` 與 `apps/admin`）。骨架のみ，無功能。版本固定於 2026-11-25，
尚未經 npm 解析驗證，待 CI 驗證。

## 1. 檔案清單

### apps/web/（ETHER 一般 Web 客戶端骨架）

| 路徑 | 用途 |
|---|---|
| `apps/web/package.json` | 固定版本依賴、scripts、`engines`（Node `>= 20.0.0`、npm `>= 10.0.0`）。 |
| `apps/web/tsconfig.json` | 嚴格模式全開（取捨見檔頭註解與 §3）。 |
| `apps/web/vite.config.ts` | 最小 Vite＋React 設定；無 proxy、無環境變數；dev port 用 Vite 預設。 |
| `apps/web/index.html` | 骨架 HTML 入口，掛載 `#root`，載入 `/src/main.tsx`。 |
| `apps/web/src/main.tsx` | React 掛載入口（StrictMode）；`#root` 缺失時拋錯，不靜默失敗。 |
| `apps/web/src/App.tsx` | 僅渲染骨架佔位頁；無按鈕、無假資料、無 API 呼叫。 |
| `apps/web/src/vite-env.d.ts` | Vite client 型別參照。 |
| `apps/web/src/styles/tokens.css` | 僅含 DESIGN_TOKENS.md §1.1 的 8 個已定色碼變數；其餘皆註解待定。 |
| `apps/web/eslint.config.js` | ESLint flat config（固定版本對應）。 |
| `apps/web/.prettierrc` | Prettier 設定（固定版本對應）。 |
| `apps/web/.gitignore` | `node_modules/`、`dist/`。 |
| `apps/web/README.md` | 建置／啟動、狀態、缺口清單。 |

### apps/admin/（ETHER Web 管理後台骨架）

| 路徑 | 用途 |
|---|---|
| `apps/admin/package.json` | 與 web 相同版本組合；套件名 `ether-admin`；scripts 相同。 |
| `apps/admin/tsconfig.json` | 與 web 相同嚴格度；管理後台程式碼不享較寬鬆檢查。 |
| `apps/admin/vite.config.ts` | 最小 Vite＋React 設定；dev port 5174（僅本地並行慣例，待確認）。 |
| `apps/admin/index.html` | 骨架 HTML 入口，標題為 ETHER Admin。 |
| `apps/admin/src/main.tsx` | 與 web 相同掛載邏輯。 |
| `apps/admin/src/App.tsx` | 僅渲染骨架佔位頁；另註明管理員認證與權限本階段不實作、絕不由前端判斷。 |
| `apps/admin/src/vite-env.d.ts` | Vite client 型別參照。 |
| `apps/admin/src/styles/tokens.css` | 與 web 相同的 8 個已定色碼變數獨立副本；另註明共用機制待決。 |
| `apps/admin/eslint.config.js` | 與 web 相同的 ESLint flat config。 |
| `apps/admin/.prettierrc` | 與 web 相同的 Prettier 設定。 |
| `apps/admin/.gitignore` | `node_modules/`、`dist/`。 |
| `apps/admin/README.md` | 建置／啟動、狀態、缺口清單，含管理員認證與權限聲明（Phase 08／11）。 |

### 本筆記

| 路徑 | 用途 |
|---|---|
| `docs/phase01/react-notes.md` | 本文件：React 軌道筆記。 |

未建立：`.github/`（禁止）、`packages/`（§26 未定義，禁止自行發明）、任何元件、
路由、狀態管理、API 客戶端、Mock 後端、`.env` 檔案。

## 2. 版本固定依據

- 版本以撰寫當日（2026-11-25）所知的穩定版為準，全部寫死 exact 版本，不用 `^` 浮動：
  React／React DOM `19.1.1`、TypeScript `5.8.3`、Vite `6.3.5`、
  `@vitejs/plugin-react` `4.4.1`、ESLint `9.28.0`、`@eslint/js` `9.28.0`、
  `typescript-eslint` `8.33.0`、Prettier `3.5.3`、`@types/react`／`@types/react-dom` `19.1.0`。
- 上述版本尚未經 npm 解析驗證，待 CI 驗證；本機未執行 `npm install`。
- `engines` 只寫下限（Node `>= 20.0.0`、npm `>= 10.0.0`），方向與本機 Node v24.22.1
  相容，不寫上限、不宣稱已在任何版本上驗證通過；待 CI 驗證。
- 兩 app 使用同一組版本，避免同一 Monorepo 內 React／工具鏈版本分叉；若 CI 發現任一
  版本無法解析，兩 app 須同步修正並更新本節。

## 3. TypeScript 嚴格選項與取捨

全開：`strict`、`noUncheckedIndexedAccess`、`exactOptionalPropertyTypes`、
`noPropertyAccessFromIndexSignature`、`noFallthroughCasesInSwitch`、`noImplicitOverride`、
`noImplicitReturns`、`noUnusedLocals`、`noUnusedParameters`、
`forceConsistentCasingInFileNames`、`useUnknownInCatchVariables`，另加
`isolatedModules`、`erasableSyntaxOnly`（配合 Vite／esbuild 單檔轉譯，避免僅型別語法
在執行期殘留）、`allowImportingTsExtensions`。

取捨說明：

- `exactOptionalPropertyTypes` 與 `noPropertyAccessFromIndexSignature` 會讓部分慣用寫法
  更囉嗦（可選屬性不得賦 `undefined`、索引簽名須用括號存取）；取捨為寧嚴勿寬，
  骨架程式碼量極小，嚴格成本可忽略，後續階段可由協調員決定是否放寬，但放寬須有紀錄。
- `noUnusedLocals`／`noUnusedParameters` 開啟：未用變數不得以底線前綴以外的靜默方式留存；
  骨架階段即維持零未用程式碼。
- `skipLibCheck: true`：僅跳過宣告檔檢查，加快型別檢查；不放寬自有程式碼檢查。
- 無 `paths` 別名：§26 未定義共用套件目錄，不發明跨 app 匯入別名（見 §5 缺口）。

## 4. Web 與 Admin 的差異說明

| 面向 | apps/web | apps/admin |
|---|---|---|
| 定位 | 一般 Web 客戶端（後續：聊天／好友／群組等，Phase 10） | 管理後台（後續：儀表板／管理／審核／監控，§19） |
| 介面方向 | 響應式／PWA／鍵盤（PLATFORM_MATRIX §1.3） | 表格／篩選／圖表／審核（PLATFORM_MATRIX §1.4），不得複製聊天介面 |
| 認證 | 一般使用者帳號（後續階段，Keycloak 方向） | 獨立管理員認證與權限（Phase 08／11）；權限以後端強制，絕不由前端判斷 |
| Dev port | Vite 預設（5173） | 5174（僅本地並行慣例，待協調員／CI 確認，非部署決定） |
| 佔位頁文字 | Web 骨架說明＋文件指向 | Admin 骨架說明＋管理員權限聲明＋文件指向 |
| 依賴／嚴格度／lint／格式 | 完全相同 | 完全相同（刻意不放寬） |
| tokens.css | 8 色碼變數 | 8 色碼變數獨立副本＋共用機制待決註解 |

相同處多於差異處是刻意的：Phase 01 只建骨架，兩 app 的差異在功能階段才會展開；
本階段不得以假儀表板、假聊天畫面預演差異。

## 5. 已知缺口（TODO）

1. 全部功能頁、路由、狀態管理、API／WebSocket 呼叫：未實作。
2. 元件庫（Web 與 Admin 各自的視覺一致元件）：未建立，Phase 02 工作。
3. Design Tokens：除 §1.1 的 8 色碼外，主題映射、字體、間距、圓角、陰影、動效、
   圖示、z-index 皆待 Phase 02 定稿；字體選用評估中，不得預寫。
4. Web／Admin／Flutter 三端 token 共用機制：SPEC-SOURCE.md §26 未定義共用套件目錄，
   不得自行發明 `packages/` 共享目錄；目前 `tokens.css` 為兩份獨立副本，同步靠人工
   核對，漂移風險存在，記為 Phase 02 待決缺口（須由協調員／Phase 02 決定共用機制，
   再回頭消除副本）。
5. Admin 管理員認證與權限設計：Phase 08／11 工作，本階段不實作。
6. PWA、響應式版型、鍵盤操作、Admin 表格／圖表／審核流程：皆未實作。
7. CI 驗證（npm 解析、建置、lint、型別檢查）：未跑過，待 CI。
8. 品牌素材（Logo、App Icon）：未完成；品牌對外使用前檢查（商標／網域／商店名稱）
   皆為阻塞項，不得宣稱品牌素材已完成。

## 6. 阻塞項

- CI 驗證環境尚未跑過；本機不跑 `npm install`，不得宣稱建置通過。
- 共用 token 機制、字體選用、主題映射、Admin 認證設計、dev port 並行慣例，
  皆待協調員／後續階段決定。
- 硬規則：未建立 `.github/`、未 push、未建 repo／PR／Release；未碰
  `~/workspace/messenger-app/`；未修改任何已批准 Phase 00 文件（`docs/` 既有文件、
  `SPEC-SOURCE.md`、`.env.example` 皆未改；`docker-compose.yml` 在工作目錄中不存在，
  未新建）；未寫假資料、Mock 後端、無作用按鈕；未寫入任何憑證／金鑰。

## 7. 靜態驗證結果

（由 React 軌工人於交付前執行，僅靜態檢查，未跑 npm／建置；結果摘要如下，詳見回報。）

- JSON 格式檢查：`package.json`、`tsconfig.json`、`.prettierrc`（兩 app）皆可解析；
  `tsconfig.json` 含註解，以 JSONC 方式解析（TypeScript 接受註解）。
- 檔案引用一致性：`index.html` → `/src/main.tsx`；`main.tsx` → `./App` 與
  `./styles/tokens.css`；`vite.config.ts` → `@vitejs/plugin-react`；
  `package.json` scripts 參照的 `tsc`／`vite`／`eslint`／`prettier` 皆在依賴中。
- 色碼逐字核對：兩份 `tokens.css` 的 8 個色碼與 DESIGN_TOKENS.md §1.1 逐字一致；
  兩 app 內無其他十六進位色碼；無元件內寫死色碼（本階段無元件）。
- Secrets 掃描：兩 app 內無金鑰／憑證／Token 字面值，無 `.env` 檔案，無 API 呼叫
  （無 `fetch`／`axios`／`XMLHttpRequest`／`WebSocket`）。
- 禁止物掃描：無 `<button`、無假儀表板數字、無 Mock／假資料字面值；佔位頁僅含文字。
