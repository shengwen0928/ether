module ether.local/api

// go directive 於 2026-11-25 固定為 1.25，與 services/api/Dockerfile 的
// golang:1.25.0-alpine3.21 建置映像一致。升級須經審查，不得改為浮動版本。
// 暫定 module path：ether.local 為佔位符，待 repo 建立後修訂。
// 本機 VM 無 go 工具鏈，編譯與測試皆為「待 CI 驗證」，不得宣稱為已通過。
go 1.25.0

require (
	github.com/lib/pq v1.12.3
	github.com/meilisearch/meilisearch-go v0.37.0
	github.com/nats-io/nats.go v1.52.0
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
