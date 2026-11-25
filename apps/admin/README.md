# ETHER Admin — Phase 01 骨架

狀態：骨架，尚無功能，待 CI 驗證。

## 版本固定聲明

版本固定於 2026-11-25，尚未經 npm 解析驗證，待 CI 驗證。`package.json`
內 React、TypeScript、Vite、eslint、prettier 及相關套件皆為固定版本（exact
版本，不用 `^` 浮動）。本機未執行 `npm install`（避免產生 `node_modules`
與未授權的網路解析），一切建置／啟動結果皆待 CI 驗證，不得宣稱為已驗證。

## 這是什麼

ETHER Web 管理後台（React＋TypeScript＋Vite）的 Phase 01
最小可啟動骨架，依 `SPEC-SOURCE.md` §26（`apps/admin`）建立，獨立於一般 Web
客戶端（`apps/web`）。`src/App.tsx` 只渲染一個骨架佔位頁，文字說明「Phase 01
骨架，尚無功能」；沒有按鈕、沒有假資料、沒有假儀表板數字，不呼叫任何 API。

後續方向（皆未實作，僅供對照）：`SPEC-SOURCE.md` §19（Web 管理後台：儀表板、使用者管理、
群組社群管理、貼圖內容管理、系統管理）、`docs/PLATFORM_MATRIX.md` §1.4（Admin：
表格／篩選／圖表／審核，獨立認證與權限，不得複製一般聊天介面）、`docs/ROADMAP.md`
§3.11（管理後台分工：工作項分散於 Phase 08／10／11，是否單列階段待審查確認）。

## 管理員認證與權限（重要）

管理員認證與權限是 Phase 08／11 工作，本階段不實作。管理員權限絕不由前端判斷；
權限以後端強制為準（`SPEC-SOURCE.md` §27 規則 12、§19）。本骨架不含任何登入畫面、
角色判斷、權限分支，不得以此骨架冒充管理功能。

## 前置工具鏈

- Node.js `>= 20.0.0`（`engines` 下限方向，與本機 Node v24 相容；待 CI 驗證）。
- npm `>= 10.0.0`。
- 本機實測環境記載（`SPEC-SOURCE.md` 環境實情）：本機 VM 有 Node v24.22.1、npm；
  無 flutter、無 go、無 docker；iOS 建置須走 macOS CI。

## 如何建置／啟動（待 CI 驗證）

```sh
cd apps/admin
npm install
npm run dev      # Vite 開發伺服器（port 5174，見 vite.config.ts；與 web 預設 port 錯開）
npm run build    # tsc --noEmit && vite build
npm run preview  # 預覽 production build
npm run lint     # eslint
npm run typecheck # tsc --noEmit
npm run format   # prettier --check .
```

環境變數：本階段無環境變數，無 `.env` 檔案，不得把憑證、金鑰寫進程式或文件。
後續 API 位址、管理員認證設定等環境變數為 TODO，待後續階段與協調員定稿後再以佔位符形式加入。

## 檔案說明

- `package.json`：固定版本依賴與 scripts；`engines` 見上。
- `tsconfig.json`：嚴格模式全開（`strict`、`noUncheckedIndexedAccess` 等，取捨見檔頭註解）；
  與 `apps/web` 相同嚴格度，管理後台程式碼不享較寬鬆檢查。
- `vite.config.ts`：最小 Vite＋React 設定；dev port 5174（僅本地並行慣例，待協調員／CI 確認）；
  無 proxy、無環境變數。
- `index.html`、`src/main.tsx`、`src/App.tsx`：最小可啟動骨架與佔位頁。
- `src/vite-env.d.ts`：Vite client 型別參照。
- `src/styles/tokens.css`：CSS 變數只寫 `docs/brand/DESIGN_TOKENS.md` §1.1 的 8
  個已定色碼；其餘 token 一律以註解標「待 Phase 02 定稿」，不得編造。
- `eslint.config.js`、`.prettierrc`：固定版本對應的 lint／格式設定。
- `.gitignore`：`node_modules/`、`dist/`。

## 已知缺口（TODO，皆未實作）

1. 全部功能頁（儀表板、使用者／群組／社群／貼圖／系統管理、檢舉審核、操作稽核）：
   未實作，待後續階段。
2. 管理員認證、權限模型、後端權限強制：未實作（Phase 08／11 工作）；前端絕不判斷權限。
3. 元件庫（表格／篩選／圖表／審核元件）：未建立（Phase 02 設計系統工作）；本階段根本不建元件，
   不得在元件內寫死色碼。
4. Design Tokens：除 §1.1 的 8 個色碼外，Dark／Light 主題映射、字體、間距、圓角、
   陰影、動效、圖示、z-index 皆待 Phase 02 定稿；字體選用評估中，不得預寫字體決定。
5. Web／Admin／Flutter 三端 token 共用機制：`SPEC-SOURCE.md` §26 未定義共用套件目錄，
   不得自行發明 `packages/` 共享目錄；本 app 的 `tokens.css` 為獨立副本，共用機制記為
   Phase 02 待決缺口。
6. 品牌素材（Logo、App Icon 等）：未完成，不得宣稱已完成；品牌對外使用前檢查清單
   （商標、網域、商店名稱等）皆為阻塞項，見 `docs/brand/BRAND_GUIDELINES.md` §17。
7. 版本解析、建置、lint、型別檢查：皆待 CI 驗證；本機只做靜態檢查（見
   `docs/phase01/react-notes.md`）。

## 阻塞項

- CI 驗證環境（npm 解析、建置、lint）尚未跑過；本機不跑 `npm install`。
- 管理員認證與權限設計、共用 token 機制、字體選用、主題映射皆待後續階段決定。
- 不得建立 `.github/` 目錄、不得 push、不得建 repo／PR／Release；成果只留本機。
