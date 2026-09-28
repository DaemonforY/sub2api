package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildGatewayRequestLogsWhere_Empty(t *testing.T) {
	where, args := buildGatewayRequestLogsWhere(&service.GatewayRequestLogFilter{})
	require.Empty(t, where)
	require.Empty(t, args)
}

func TestBuildGatewayRequestLogsWhere_AllFilters(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	uid, kid := int64(7), int64(42)
	success := false
	code := 401
	where, args := buildGatewayRequestLogsWhere(&service.GatewayRequestLogFilter{
		StartTime:  &start,
		UserID:     &uid,
		APIKeyID:   &kid,
		Success:    &success,
		StatusCode: &code,
		Method:     "post",
		Query:      "50%_off",
	})
	require.Equal(t,
		"WHERE l.created_at >= $1 AND l.user_id = $2 AND l.api_key_id = $3 AND l.success = $4 AND l.status_code = $5 AND l.method = $6 AND "+
			"(l.url ILIKE $7 OR l.api_key ILIKE $7 OR l.model ILIKE $7 OR l.client_ip ILIKE $7 OR u.email ILIKE $7 OR k.name ILIKE $7)",
		where)
	require.Equal(t, []any{start, uid, kid, false, 401, "POST", `%50\%\_off%`}, args)
}
