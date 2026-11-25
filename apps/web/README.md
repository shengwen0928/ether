# ETHER Web — Phase 01 骨架

狀態：骨架，尚無功能，待 CI 驗證。

## 版本固定聲明

版本固定於 2026-11-25，尚未經 npm 解析驗證，待 CI 驗證。`package.json`
內 React、TypeScript、Vite、eslint、prettier 及相關套件皆為固定版本（exact
版本，不用 `^` 浮動）。本機未執行 `npm install`（避免產生 `node_modules`
與未授權的網路解析），一切建置／啟動結果皆待 CI 驗證，不得宣稱為已驗證。

## 這是什麼

ETHER 一般 Web 客戶端（React＋TypeScript＋Vite）的 Phase 01
最小可啟動骨架，依 `SPEC-SOURCE.md` §26（`apps/web`）建立。`src/App.tsx`
只渲染一個骨架佔位頁，文字說明「Phase 01 骨架，尚無功能」；沒有按鈕、沒有假資料、
沒有假聊天畫面，不呼叫任何 API。

後續方向（皆未實作，僅供對照）：`docs/PLATFORM_MATRIX.md` §1.3（響應式／PWA／鍵盤）、
`SPEC-SOURCE.md` §03（視覺設計規範）、§07–§09（聊天／群組／社群，後續階段）。

## 前置工具鏈

- Node.js `>= 20.0.0`（`engines` 下限方向，與本機 Node v24 相容；待 CI 驗證）。
- npm `>= 10.0.0`。
- 本機實測環境記載（`SPEC-SOURCE.md` 環境實情）：本機 VM 有 Node v24.22.1、npm；
  無 flutter、無 go、無 docker；iOS 建置須走 macOS CI。

## 如何建置／啟動（待 CI 驗證）

```sh
cd apps/web
npm install
npm run dev      # Vite 開發伺服器（預設 port，見 vite.config.ts）
npm run build    # tsc --noEmit && vite build
npm run preview  # 預覽 production build
npm run lint     # eslint
npm run typecheck # tsc --noEmit
npm run format   # prettier --check .
```

環境變數（Vite `VITE_` 前綴，建置期讀入，不得寫入憑證、金鑰或 Token）：

| 變數 | 說明 |
|---|---|
| `VITE_DATA_MODE` | `demo`（預設，展示資料）或 `api`（連接 `services/api`）。 |
| `VITE_API_BASE_URL` | API 來源（空字串＝同源）。 |
| `VITE_KEYCLOAK_ISSUER` | Keycloak 發行者位址，如 `http://localhost:8080/realms/ether`。三者缺一即停用 OIDC 登入路徑。 |
| `VITE_KEYCLOAK_CLIENT_ID` | Public client id，如 `ether-web`（無 secret）。 |
| `VITE_KEYCLOAK_REDIRECT_URI` | 須與 Keycloak 註冊的 redirect URI 一致，如 `http://localhost:5173/`。 |
| `VITE_ALLOW_DEV_TOKEN_LOGIN` | `true` 時在 OIDC 已設定下仍顯示開發者 Token 表單（僅開發用）。 |

Token 只存記憶體（見 `src/data/oidc.ts` 檔頭），分頁關閉或重新整理後須重新登入。

## 檔案說明

- `package.json`：固定版本依賴與 scripts；`engines` 見上。
- `tsconfig.json`：嚴格模式全開（`strict`、`noUncheckedIndexedAccess` 等，取捨見檔頭註解）。
- `vite.config.ts`：最小 Vite＋React 設定；無 proxy、無環境變數。
- `index.html`、`src/main.tsx`、`src/App.tsx`：最小可啟動骨架與佔位頁。
- `src/vite-env.d.ts`：Vite client 型別參照。
- `src/styles/tokens.css`：CSS 變數只寫 `docs/brand/DESIGN_TOKENS.md` §1.1 的 8
  個已定色碼；其餘 token 一律以註解標「待 Phase 02 定稿」，不得編造。
- `eslint.config.js`、`.prettierrc`：固定版本對應的 lint／格式設定。
- `.gitignore`：`node_modules/`、`dist/`。

## 已知缺口（TODO，皆未實作）

1. 全部功能頁、路由、狀態管理、API 呼叫、WebSocket：未實作，待後續階段。
2. 元件庫：未建立（Phase 02 設計系統工作）；本階段根本不建元件，不得在元件內寫死色碼。
3. Design Tokens：除 §1.1 的 8 個色碼外，Dark／Light 主題映射、字體、間距、圓角、
   陰影、動效、圖示、z-index 皆待 Phase 02 定稿；字體選用評估中，不得預寫字體決定。
4. Web／Admin／Flutter 三端 token 共用機制：`SPEC-SOURCE.md` §26 未定義共用套件目錄，
   不得自行發明 `packages/` 共享目錄；本 app 的 `tokens.css` 為獨立副本，共用機制記為
   Phase 02 待決缺口。
5. PWA（Service Worker、Web Push）、響應式版型、鍵盤操作：未實作，待後續階段與實測。
6. 品牌素材（Logo、App Icon 等）：未完成，不得宣稱已完成；品牌對外使用前檢查清單
   （商標、網域、商店名稱等）皆為阻塞項，見 `docs/brand/BRAND_GUIDELINES.md` §17。
7. 版本解析、建置、lint、型別檢查：皆待 CI 驗證；本機只做靜態檢查（見
   `docs/phase01/react-notes.md`）。

## 阻塞項

- CI 驗證環境（npm 解析、建置、lint）尚未跑過；本機不跑 `npm install`。
- 共用 token 機制、字體選用、主題映射皆待 Phase 02 決定。
- 不得建立 `.github/` 目錄、不得 push、不得建 repo／PR／Release；成果只留本機。
