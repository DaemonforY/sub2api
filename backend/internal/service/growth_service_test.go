//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEduSuffixes(t *testing.T) {
	require.Equal(t,
		[]string{"edu.cn", "ac.uk", "mails.tsinghua.edu.cn"},
		normalizeEduSuffixes([]string{" @EDU.cn ", "*.ac.uk", ".edu.cn", "mails.tsinghua.edu.cn", "", "localhost", "bad/x.com", "a@b.com"}),
	)
}

func TestIsSchoolEmailDomain(t *testing.T) {
	suffixes := []string{"edu.cn", "ac.uk"}
	require.True(t, isSchoolEmailDomain("pku.edu.cn", suffixes))
	require.True(t, isSchoolEmailDomain("mails.tsinghua.edu.cn", suffixes))
	require.True(t, isSchoolEmailDomain("edu.cn", suffixes))
	require.True(t, isSchoolEmailDomain("OX.AC.UK", suffixes))
	require.False(t, isSchoolEmailDomain("fakeedu.cn", suffixes), "must match on a label boundary")
	require.False(t, isSchoolEmailDomain("qq.com", suffixes))
}

func TestParseGrowthSettingsDefaultsAndClamps(t *testing.T) {
	s := parseGrowthSettings(map[string]string{})
	require.Zero(t, s.InviteeBonusRatePercent)
	require.True(t, s.LeaderboardEnabled, "leaderboard is on unless turned off")
	require.False(t, s.EduVerifyEnabled)
	require.Equal(t, []string{"edu.cn"}, s.EduEmailSuffixes)

	s = parseGrowthSettings(map[string]string{
		SettingKeyGrowthInviteeBonusRate:   "150",
		SettingKeyGrowthInviteeBonusCap:    "-3",
		SettingKeyGrowthLeaderboardEnabled: "false",
		SettingKeyEduSubscriptionDiscount:  "95",
		SettingKeyEduEmailSuffixes:         `["@ac.uk"]`,
	})
	require.Equal(t, 100.0, s.InviteeBonusRatePercent)
	require.Zero(t, s.InviteeBonusCap)
	require.False(t, s.LeaderboardEnabled)
	require.Equal(t, 90.0, s.EduDiscountPercent)
	require.Equal(t, []string{"ac.uk"}, s.EduEmailSuffixes)
}

func TestLeaderboardPeriod(t *testing.T) {
	now := time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)
	name, start, end := leaderboardPeriod("", now)
	require.Equal(t, "month", name)
	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), *start)
	require.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), *end)

	name, start, end = leaderboardPeriod("last_month", now)
	require.Equal(t, "last_month", name)
	require.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), *start)
	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), *end)

	name, start, end = leaderboardPeriod("all", now)
	require.Equal(t, "all", name)
	require.Nil(t, start)
	require.Nil(t, end)
}

func TestPublicLeaderboardEntryHidesIdentityAndMoney(t *testing.T) {
	row := InviteLeaderboardEntry{Rank: 1, UserID: 7, Email: "alice@example.com", Username: "alice", InvitedCount: 3, PayingInvitees: 2, InviteePaid: 80, RebateAccrued: 16}
	e := publicLeaderboardEntry(row, 7)
	require.NotContains(t, e.DisplayName, "alice")
	require.Zero(t, e.UserID)
	require.Empty(t, e.Email)
	require.Zero(t, e.InviteePaid)
	require.Zero(t, e.RebateAccrued)
	require.True(t, e.IsCurrentUser)

	row.Username = ""
	e = publicLeaderboardEntry(row, 8)
	require.NotContains(t, e.DisplayName, "alice@example.com")
	require.False(t, e.IsCurrentUser)
}
