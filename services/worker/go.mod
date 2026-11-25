module ether.local/worker

// go directive 於 2026-11-25 固定為 1.25，與 services/worker/Dockerfile 的
// golang:1.25.0-alpine3.21 建置映像一致。升級須經審查，不得改為浮動版本。
// 暫定 module path：ether.local 為佔位符，待 repo 建立後修訂。
// 本機 VM 無 go 工具鏈，編譯與測試皆為「待 CI 驗證」，不得宣稱為已通過。
go 1.25
