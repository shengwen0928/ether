// Package migrations 內嵌 services/api/migrations 下的 SQL 檔案，
// 供 migrator（internal/postgres）套用，並提供不依賴資料庫的
// 靜態驗證：命名、編號連續、up／down 配對、內容非空、up 檔
// 不得含破壞性陳述。
//
// 內嵌檔案的增刪即 migration 的增刪：新增 migration 時在此目錄
// 新增一對 up／down 檔案即可，不得另寫清單（清單即目錄本身）。
package migrations

import (
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

//go:embed *.up.sql *.down.sql
var FS embed.FS

// Migration 是一個版本的 up／down SQL。
type Migration struct {
	Version string // 六位數字，如 000002。
	Name    string // 底線分隔的簡短描述，如 identity。
	Up      string
	Down    string
}

var fileNameRe = regexp.MustCompile(`^([0-9]{6})_([a-z0-9_]+)\.(up|down)\.sql$`)

// List 讀出內嵌的全部 migration，依版本遞增排序。
func List() ([]Migration, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("migrations: 讀取內嵌目錄失敗: %w", err)
	}
	byVersion := map[string]*Migration{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migrations: 檔名不符合慣例: %s", e.Name())
		}
		version, name, direction := m[1], m[2], m[3]
		content, err := FS.ReadFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("migrations: 讀取 %s 失敗: %w", e.Name(), err)
		}
		mig, ok := byVersion[version]
		if !ok {
			mig = &Migration{Version: version, Name: name}
			byVersion[version] = mig
		}
		if mig.Name != name {
			return nil, fmt.Errorf("migrations: 版本 %s 的 up／down 名稱不一致: %s 與 %s", version, mig.Name, name)
		}
		if direction == "up" {
			mig.Up = string(content)
		} else {
			mig.Down = string(content)
		}
	}
	out := make([]Migration, 0, len(byVersion))
	for _, mig := range byVersion {
		out = append(out, *mig)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// Validate 靜態驗證全部 migration（不連接資料庫）：
//   - 版本自 000001 起連續遞增，不得跳號、不得重複；
//   - 每個版本皆有 up 與 down，且內容非空；
//   - up 檔不得含 DROP TABLE／DROP COLUMN（破壞性變更須以新
//     migration 修正，不得藏在新增檔案的 up 語句中）；
//   - up 檔須含至少一個 CREATE TABLE 或 ALTER TABLE。
func Validate() error {
	list, err := List()
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return fmt.Errorf("migrations: 沒有任何 migration")
	}
	for i, mig := range list {
		want := fmt.Sprintf("%06d", i+1)
		if mig.Version != want {
			return fmt.Errorf("migrations: 版本不連續：第 %d 個為 %s，應為 %s", i+1, mig.Version, want)
		}
		if strings.TrimSpace(mig.Up) == "" {
			return fmt.Errorf("migrations: %s 缺少 up.sql 內容", mig.Version)
		}
		if strings.TrimSpace(mig.Down) == "" {
			return fmt.Errorf("migrations: %s 缺少 down.sql 內容", mig.Version)
		}
		upper := strings.ToUpper(mig.Up)
		if strings.Contains(upper, "DROP TABLE") || strings.Contains(upper, "DROP COLUMN") {
			return fmt.Errorf("migrations: %s 的 up.sql 不得含 DROP TABLE／DROP COLUMN", mig.Version)
		}
		if !strings.Contains(upper, "CREATE TABLE") && !strings.Contains(upper, "ALTER TABLE") {
			return fmt.Errorf("migrations: %s 的 up.sql 須含 CREATE TABLE 或 ALTER TABLE", mig.Version)
		}
	}
	return nil
}
