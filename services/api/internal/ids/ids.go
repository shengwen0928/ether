// Package ids 提供 RFC 4122 第 4 版 UUID 字串產生器（純標準庫）。
//
// 各領域模組以此產生主鍵，避免在多個套件重複實作亂數 UUID。
// 產生失敗（亂數來源不可用）時回傳空字串，呼叫方不得以空字串
// 作為識別碼寫入儲存。
package ids

import (
	"crypto/rand"
	"fmt"
)

// New 產生一個隨機 UUID v4 字串（小寫、連字號分隔）。
func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
