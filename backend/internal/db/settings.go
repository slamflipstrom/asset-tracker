package db

import "context"

func (d *DB) FetchAppSettings(ctx context.Context) (AppSettings, error) {
	row := d.pool.QueryRow(ctx, `
		select min_refresh_interval_sec, max_refresh_interval_sec
		from public.app_settings
		where id = 1
	`)
	var settings AppSettings
	if err := row.Scan(&settings.MinRefreshIntervalSec, &settings.MaxRefreshIntervalSec); err != nil {
		return settings, err
	}
	return settings, nil
}

func (d *DB) FetchUserSettings(ctx context.Context, userID string) (UserSettings, error) {
	row := d.pool.QueryRow(ctx, `
		select user_id, refresh_interval_sec, created_at, updated_at
		from public.user_settings
		where user_id = $1
	`, userID)
	var settings UserSettings
	if err := row.Scan(&settings.UserID, &settings.RefreshIntervalSec, &settings.CreatedAt, &settings.UpdatedAt); err != nil {
		return settings, err
	}
	return settings, nil
}
