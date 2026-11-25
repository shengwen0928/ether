# 安全儲存（flutter_secure_storage）— Phase 01 佔位說明

狀態：Phase 01 空骨架。本階段不呼叫 `flutter_secure_storage`，不讀寫任何憑證。

## 宣告

`flutter_secure_storage` 僅在 `apps/mobile/pubspec.yaml` 宣告（版本固定，待 CI 解析驗證）。
本目錄不含任何儲存實作程式碼。

## 用途（延至 Phase 03）

- 用途：Keycloak token 儲存（Access／Refresh Token）。
- 階段：Phase 03 身分帳號（手冊 E），隨 Keycloak 接線一併實作。
- 依據：`SPEC-SOURCE.md` §05／手冊 E、`docs/architecture/SYSTEM_ARCHITECTURE.md` §3.2、
  `docs/decisions/ADR-010-Keycloak身分架構.md`。

## 禁止事項

- 不得把憑證、金鑰、Token 明文寫進程式碼、文件或版本控管。
- 不得把 E2EE 金鑰明文或訊息明文寫入任何儲存。
- Token 儲存方式、Refresh Rotation、Session 撤銷等細節，待 Phase 03 定稿，不得預寫。

## 缺口

- Keycloak 配置定稿（阻塞項，見 `docs/ROADMAP.md` Phase 03 阻塞項）。
- 安全儲存的平台實作驗收（Android／iOS 真機），待 Phase 03。
