# Design System — Phase 01 佔位說明

狀態：Phase 01 空骨架。本目錄不含任何元件。

## 元件庫是 Phase 02 工作

Flutter 元件庫（與 React 元件庫視覺一致）屬 Phase 02 設計系統與品牌（手冊 D），
本階段不得建立任何元件，不得寫死色碼。

## 依據

- 視覺設計規範：`SPEC-SOURCE.md` §03（元件狀態、動畫與觸覺回饋要求）。
- 色彩 token 唯一來源：`docs/brand/DESIGN_TOKENS.md`（本階段只定了 8 個品牌色，
  其餘 token 待 Phase 02 定稿，不得編造）。
- 品牌禁止事項：`docs/brand/BRAND_GUIDELINES.md` §8（禁大量紫藍漸層、隨機發光球體、
  過多玻璃卡片、無意義裝飾線條、廉價科技裝飾；禁以 Material 3 預設外觀代替品牌設計）。
- 導覽架構：`SPEC-SOURCE.md` §04（底部五入口）。

## 規則

- Phase 02 建立元件時，一律以 `DESIGN_TOKENS.md` 為唯一色彩與字體來源，不得在元件內寫死色碼。
- 新增 token 須先更新 `DESIGN_TOKENS.md` 並經審查。
- App Icon：原稿在 `docs/brand/assets/app-icon-source.jpg`；PNG 匯出與各尺寸辨識度驗證是
  Phase 02 工作，本階段不得宣稱 launcher 圖示已完成。

## 缺口

- 全部 Design Tokens（除 8 個品牌色外）待 Phase 02 定稿。
- Logo、App Icon、字體選用，待 Phase 02（字體授權須先查證）。
