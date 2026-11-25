# ETHER Design Tokens（DESIGN_TOKENS.md）

狀態：草案／待批准。依據：品牌企劃書 §8 色彩與 `docs/brand/BRAND_GUIDELINES.md` §8。

**誠實規則：** 本文件只收錄品牌企劃書已給定的色碼；凡企劃書未給定的 token，一律標「待 Phase 02 設計系統定稿」，不得編造色碼或數值。色碼以企劃書為準，逐一核對過（見 §4）。

## 1. 色彩 Tokens

### 1.1 品牌色板（已定，企劃書 §8）

| Token | 色碼 | 用途 |
|---|---|---|
| `ether-midnight` | #101923 | 品牌深色背景 |
| `deep-nebula` | #182A3A | 深層介面 |
| `ethereal-blue` | #86B9D2 | 品牌識別色 |
| `silver-mist` | #C7D5DE | 輔助文字 |
| `pure-light` | #F4F7F9 | 明亮背景 |
| `aurora-cyan` | #65D4D6 | 互動與狀態 |
| `soft-violet` | #A99ACB | AI 與特殊功能 |
| `warm-horizon` | #D9B98C | 情感與記憶內容 |

品牌色彩不應全部同時使用；主識別以 Ether Midnight、Ethereal Blue、Pure Light 建立，其餘色彩用於功能區分與情境表達。

### 1.2 Dark Theme（深色，以 Ether Midnight 系為基調）

| Token | 值 | 來源 |
|---|---|---|
| `color.background` | #101923（`ether-midnight`） | 已定 |
| `color.surface` | #182A3A（`deep-nebula`） | 已定 |
| `color.surface-elevated` | 待 Phase 02 定稿 | 待定，不得編造 |
| `color.primary` | #86B9D2（`ethereal-blue`） | 已定 |
| `color.text-primary` | #F4F7F9（`pure-light`） | 已定 |
| `color.text-secondary` | #C7D5DE（`silver-mist`） | 已定 |
| `color.border` | 待 Phase 02 定稿 | 待定，不得編造 |
| `color.interactive` | #65D4D6（`aurora-cyan`） | 已定 |
| `color.ai` | #A99ACB（`soft-violet`） | 已定 |
| `color.memory` | #D9B98C（`warm-horizon`） | 已定 |
| `color.success`／`color.warning`／`color.error`／`color.info` | 待 Phase 02 定稿 | 待定，不得編造 |

### 1.3 Light Theme（淺色，以 Pure Light 為基調）

| Token | 值 | 來源 |
|---|---|---|
| `color.background` | #F4F7F9（`pure-light`） | 已定 |
| `color.surface` | 待 Phase 02 定稿（不得預寫為純白或其他色碼） | 待定，不得編造 |
| `color.surface-elevated` | 待 Phase 02 定稿 | 待定，不得編造 |
| `color.primary` | #86B9D2（`ethereal-blue`） | 已定 |
| `color.text-primary` | #101923（`ether-midnight`，淺色背景上的主要文字） | 已定（色板既有色） |
| `color.text-secondary` | 待 Phase 02 定稿（Silver Mist 在淺色背景對比不足，不得直接沿用為次要文字色） | 待定，不得編造 |
| `color.border` | 待 Phase 02 定稿 | 待定，不得編造 |
| `color.interactive` | #65D4D6（`aurora-cyan`） | 已定 |
| `color.ai` | #A99ACB（`soft-violet`） | 已定 |
| `color.memory` | #D9B98C（`warm-horizon`） | 已定 |
| `color.success`／`color.warning`／`color.error`／`color.info` | 待 Phase 02 定稿 | 待定，不得編造 |

### 1.4 禁止事項

- 禁止大量紫藍漸層、隨機發光球體、過多玻璃卡片、無意義裝飾線條、廉價科技裝飾。
- 禁止以 Material 3 預設外觀代替品牌設計；Material 3 僅為 Flutter 實作基底之一（見 ADR-014）。

## 2. 字體 Tokens（框架，選用待定）

| Token | 內容 | 狀態 |
|---|---|---|
| `font.family.en` | Inter／Manrope／Plus Jakarta Sans／Geist 擇一 | 評估中，待 Phase 02 定稿 |
| `font.family.zh` | Noto Sans TC／思源黑體／LINE Seed Sans TC 擇一（第三方字體須先確認授權） | 評估中，待 Phase 02 定稿 |
| `font.family.logo` | 專屬字標設計，不必直接使用 UI 字體 | 待 Phase 02 設計 |
| `font.scale`／`font.weight`／`font.line-height` | 字級層次、字重、行高 | 待 Phase 02 定稿，不得編造 |

排版原則：清楚的字級層次、舒適的行高與閱讀距離、適當留白、避免過多粗體與過度密集的資訊；品牌宣傳可用大標題與極簡構圖；產品介面以資訊清晰與操作效率優先。

## 3. 間距、圓角、陰影、動效 Tokens（框架，待定）

| Token 類別 | 狀態 |
|---|---|
| `spacing.*`（間距階梯） | 待 Phase 02 定稿，不得編造 |
| `radius.*`（圓角階梯；避免過度圓角的卡片，見原始規格 §03） | 待 Phase 02 定稿，不得編造 |
| `border.*`（邊框） | 待 Phase 02 定稿，不得編造 |
| `shadow.*`／`elevation.*`（陰影／層級） | 待 Phase 02 定稿，不得編造 |
| `motion.*`（動畫時長／曲線；互動須有一致的動畫規範） | 待 Phase 02 定稿，不得編造 |
| `icon.*`（圖示系統） | 待 Phase 02 定稿，不得編造 |
| `z-index.*` | 待 Phase 02 定稿，不得編造 |

## 4. 色碼核對紀錄

本節色碼已與使用者品牌企劃書 §8 逐一核對一致：#101923、#182A3A、#86B9D2、#C7D5DE、#F4F7F9、#65D4D6、#A99ACB、#D9B98C。核對人：品牌替換協調員；核對方式：逐字比對企劃書色碼表。

## 5. 與元件庫的關係

Phase 02（手冊 D）建立 Flutter／React 共用元件庫時，一律以本文件 tokens 為唯一色彩與字體來源，不得在元件內寫死色碼；新增 token 須先更新本文件並經審查。
