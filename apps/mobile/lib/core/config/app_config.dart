// ETHER Phase 01 骨架 — 環境設定佔位。
//
// 全部為佔位符，無真實金鑰、無真實 URL、無真實憑證。
// 真實設定一律經環境變數／CI Secrets／OpenBao 注入，不得寫入程式碼（見 SPEC-SOURCE.md §27 第 11 條）。
// 環境劃分（local／development／staging／production）見 docs/architecture/SYSTEM_ARCHITECTURE.md §6。
// TODO(Phase 01 CI): 定案環境注入方式（flavor／--dart-define／設定檔），再實作讀取。
// TODO(後續階段): Keycloak、API、LiveKit 等端點設定，待各階段定稿後再接線；本階段不得預寫。

/// 部署環境列舉（名稱取自 SYSTEM_ARCHITECTURE.md §6，不得自行增減）。
enum AppEnvironment {
  local,
  development,
  staging,
  production,
}

/// 環境設定佔位。
///
/// 本階段不讀取任何真實設定；以下預設值皆為空佔位，僅保留結構。
class AppConfig {
  const AppConfig._();

  /// 目前環境。預設為 local 佔位，注入方式待 CI 定案。
  static const AppEnvironment environment = AppEnvironment.local;

  /// API 基底 URL 佔位。無真實值；以 --dart-define 或環境變數注入，待定案。
  static const String apiBaseUrl = String.fromEnvironment(
    'ETHER_API_BASE_URL',
    defaultValue: '',
  );

  /// 是否為正式環境。佔位判斷，待環境注入定案後再實作。
  static bool get isProduction => environment == AppEnvironment.production;
}
