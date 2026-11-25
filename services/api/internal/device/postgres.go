// PostgreSQL 儲存實作（M2b-1）：device.Store。
//
// 對映規則（schema 見 migrations/000002）：user_devices 全欄位對映；
// Revoked（bool）寫入 is_revoked，RevokedAt 寫入 revoked_at（可空）。
package device

import (
	"context"
	"database/sql"

	"ether.local/api/internal/postgres"
)

// PGStore 是 Store 的 PostgreSQL 實作。
type PGStore struct{ q postgres.Querier }

// NewPGStore 建立 PGStore。
func NewPGStore(q postgres.Querier) *PGStore { return &PGStore{q: q} }

const deviceColumns = `id::text, user_id::text, device_name, platform, device_identifier_hash, last_active_at, is_trusted, is_revoked, revoked_at, created_at, updated_at`

func scanDevice(row postgres.Row) (*Device, error) {
	var d Device
	var lastActive, revokedAt sql.NullTime
	if err := row.Scan(&d.ID, &d.UserID, &d.DeviceName, &d.Platform, &d.DeviceIdentifierHash,
		&lastActive, &d.Trusted, &d.Revoked, &revokedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
		if postgres.IsNoRows(err) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	d.LastActiveAt = postgres.TimePtrFromNull(lastActive)
	d.RevokedAt = postgres.TimePtrFromNull(revokedAt)
	return &d, nil
}

func (s *PGStore) Get(ctx context.Context, id string) (*Device, error) {
	return scanDevice(s.q.QueryRowContext(ctx,
		`SELECT `+deviceColumns+` FROM user_devices WHERE id = $1::uuid`, id))
}

func (s *PGStore) ListByUser(ctx context.Context, userID string) ([]*Device, error) {
	rows, err := s.q.QueryContext(ctx,
		`SELECT `+deviceColumns+` FROM user_devices WHERE user_id = $1::uuid ORDER BY created_at ASC, id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PGStore) Create(ctx context.Context, d *Device) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO user_devices (id, user_id, device_name, platform, device_identifier_hash, last_active_at, is_trusted, is_revoked, revoked_at, created_at, updated_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		d.ID, d.UserID, d.DeviceName, d.Platform, d.DeviceIdentifierHash,
		postgres.NullTime(d.LastActiveAt), d.Trusted, d.Revoked,
		postgres.NullTime(d.RevokedAt), d.CreatedAt, d.UpdatedAt)
	return err
}

func (s *PGStore) Save(ctx context.Context, d *Device) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE user_devices SET device_name = $2, platform = $3, device_identifier_hash = $4,
		 last_active_at = $5, is_trusted = $6, is_revoked = $7, revoked_at = $8, updated_at = $9
		 WHERE id = $1::uuid`,
		d.ID, d.DeviceName, d.Platform, d.DeviceIdentifierHash,
		postgres.NullTime(d.LastActiveAt), d.Trusted, d.Revoked,
		postgres.NullTime(d.RevokedAt), d.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

var _ Store = (*PGStore)(nil)
