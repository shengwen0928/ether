// Package config 從環境變數載入 api 服務的設定。
//
// 變數名來源（不得自創衝突的變數名）：
//   - 依賴位址：由 infrastructure/docker/docker-compose.yml 的 api 服務
//     environment 區塊以服務名組裝（DATABASE_URL、VALKEY_URL、NATS_URL、
//     MINIO_ENDPOINT、MEILISEARCH_URL、KEYCLOAK_URL）。應用服務不得自行
//     假設位址（見 .env.example 末節註解）。
//   - 覆寫慣例：任一依賴位址可以 <NAME>_OVERRIDE 覆寫（例如
//     DATABASE_URL_OVERRIDE），覆寫值優先於 compose 組裝值；覆寫清單
//     的最終定稿待 Phase 01 確認（見 .env.example 末節）。
//   - 埠號：API_PORT（.env.example 末節列出的名稱，註解狀態）；未設定時
//     使用 8080，與 compose healthcheck 的埠號一致，不得以此預設值
//     冒充為已定稿的設定。
//   - 身分設定：KEYCLOAK_REALM（Realm 名）、KEYCLOAK_CLIENT_ID
//     （Client 識別碼，見 .env.example Keycloak 節）、KEYCLOAK_AUDIENCE
//     （Token aud 強制值，空值表示不強制，見 M2b Keycloak 整合文件）、
//     IDENTITY_STORE（身分儲存後端：postgres／memory，預設 postgres）。
//
// 缺值時明確報錯：Load 在任一依賴位址缺值時回傳錯誤，列出缺失的
// 變數名，不得給預設真憑證、不得以空字串靜默啟動。憑證本身一律
// 走 secrets 套件／OpenBao，不得寫入本套件或設定檔。
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config 是 api 服務的執行期設定。欄位值皆來自環境變數，不得含
// 寫死的真憑證；本階段不建立任何依賴連線（連線是 TODO）。
type Config struct {
	// Port 是 HTTP 監聽埠號（字串形式，如 "8080"）。
	Port string
	// DatabaseURL 是 PostgreSQL 連線字串（compose 組裝）。
	DatabaseURL string
	// ValkeyURL 是 Valkey 連線字串（compose 組裝）。
	ValkeyURL string
	// NATSURL 是 NATS JetStream 連線字串（compose 組裝）。
	NATSURL string
	// MinIOEndpoint 是 MinIO 端點（compose 組裝，主機:埠形式）。
	MinIOEndpoint string
	// MeilisearchURL 是 Meilisearch 位址（compose 組裝）。
	MeilisearchURL string
	// KeycloakURL 是 Keycloak 位址（compose 組裝）。
	KeycloakURL string
	// KeycloakRealm 是 Keycloak Realm 名稱（環境變數 KEYCLOAK_REALM）。
	// 暫定預設值 "ether"：Realm 隔離方式尚待定稿
	//（IDENTITY_AND_PERMISSIONS.md §12 缺口 11），本預設僅供開發
	//環境啟動，不得解讀為已定稿的 Realm 劃分；正式部署須以環境
	//變數明確指定。
	KeycloakRealm string
	// KeycloakClientID 是 Keycloak Client 識別碼（環境變數
	// KEYCLOAK_CLIENT_ID）。RS256 驗證不需 Client 識別碼；本欄位
	// 供日後授權碼流程與 Realm 範本一致性核對使用，空值允許。
	KeycloakClientID string
	// KeycloakAudience 是 Token aud 強制值（環境變數
	// KEYCLOAK_AUDIENCE）。空值表示不強制 aud（M1 預設行為）；
	// 設定值時驗證器要求 Token 的 aud（字串或陣列）包含該值。
	// 正式部署應設定為 API Client 的識別碼，見
	// docs/development/KEYCLOAK_INTEGRATION.md。
	KeycloakAudience string
	// IdentityStore 是身分儲存後端（環境變數 IDENTITY_STORE）：
	// "postgres"（預設，PostgreSQL 儲存庫，啟動時 Ping 失敗即
	// 明確報錯退出）或 "memory"（開發／測試用，資料不持久）。
	// 其他值一律由 Load 報錯，不得靜默降級。
	IdentityStore string
}

// dependencyEnv 是 compose 為 api 服務組裝的依賴位址變數名（依
// docker-compose.yml 的 environment 區塊順序）。
var dependencyEnv = []struct {
	field string
	env   string
}{
	{"DatabaseURL", "DATABASE_URL"},
	{"ValkeyURL", "VALKEY_URL"},
	{"NATSURL", "NATS_URL"},
	{"MinIOEndpoint", "MINIO_ENDPOINT"},
	{"MeilisearchURL", "MEILISEARCH_URL"},
	{"KeycloakURL", "KEYCLOAK_URL"},
}

// resolve 讀取一個依賴位址：<NAME>_OVERRIDE 優先於 <NAME>。
// 兩者皆空時回傳空字串，由 Load 彙總報錯。
func resolve(name string) string {
	if v := strings.TrimSpace(os.Getenv(name + "_OVERRIDE")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(name))
}

// Load 從環境變數載入 Config。任一依賴位址缺值時回傳錯誤，
// 不得回傳部分設定讓呼叫方以缺失的位址啟動。
func Load() (Config, error) {
	cfg := Config{
		Port: strings.TrimSpace(os.Getenv("API_PORT")),
	}
	if cfg.Port == "" {
		// 預設值等於 compose healthcheck 埠號；正式的埠號設定方式
		// 待 Phase 01 定案，不得以此預設冒充定稿。
		cfg.Port = "8080"
	}

	values := map[string]string{}
	var missing []string
	for _, d := range dependencyEnv {
		v := resolve(d.env)
		values[d.field] = v
		if v == "" {
			missing = append(missing, d.env+"（或 "+d.env+"_OVERRIDE）")
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: 缺少必要的環境變數: %s",
			strings.Join(missing, "、"))
	}

	cfg.DatabaseURL = values["DatabaseURL"]
	cfg.ValkeyURL = values["ValkeyURL"]
	cfg.NATSURL = values["NATSURL"]
	cfg.MinIOEndpoint = values["MinIOEndpoint"]
	cfg.MeilisearchURL = values["MeilisearchURL"]
	cfg.KeycloakURL = values["KeycloakURL"]
	cfg.KeycloakRealm = strings.TrimSpace(os.Getenv("KEYCLOAK_REALM"))
	if cfg.KeycloakRealm == "" {
		// 暫定預設值（見 Config.KeycloakRealm 註解），僅供開發啟動。
		cfg.KeycloakRealm = "ether"
	}
	cfg.KeycloakClientID = strings.TrimSpace(os.Getenv("KEYCLOAK_CLIENT_ID"))
	cfg.KeycloakAudience = strings.TrimSpace(os.Getenv("KEYCLOAK_AUDIENCE"))
	cfg.IdentityStore = strings.TrimSpace(os.Getenv("IDENTITY_STORE"))
	if cfg.IdentityStore == "" {
		cfg.IdentityStore = "postgres"
	}
	switch cfg.IdentityStore {
	case "postgres", "memory":
	default:
		return Config{}, fmt.Errorf("config: IDENTITY_STORE 須為 postgres 或 memory，現為 %q", cfg.IdentityStore)
	}
	return cfg, nil
}
