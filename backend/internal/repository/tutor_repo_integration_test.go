//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestTutorRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "tutor-" + suffix + "@tutor.test", Username: "tut" + suffix[len(suffix)-4:]})
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM tutors WHERE user_id = $1`, user.ID) })
	repo := NewTutorRepository(integrationDB)

	tu := &service.Tutor{UserID: user.ID, KeyID: 9, Name: "数学助教", Template: "qa", AnswerMode: "guide", ShareCode: "c" + suffix[len(suffix)-7:],
		PassCode: "8023", PerStudentDay: 20, DailyCap: 300, Enabled: true}
	require.NoError(t, repo.Create(ctx, tu))
	got, err := repo.GetByCode(ctx, tu.ShareCode)
	require.NoError(t, err)
	require.Equal(t, "数学助教", got.Name)
	other, err := repo.Get(ctx, user.ID+100000, tu.ID)
	require.NoError(t, err)
	require.Nil(t, other)

	tu.Name, tu.Enabled = "改名了", false
	require.NoError(t, repo.Update(ctx, tu))
	got, _ = repo.Get(ctx, user.ID, tu.ID)
	require.Equal(t, "改名了", got.Name)
	require.False(t, got.Enabled)

	m, err := repo.AddMaterial(ctx, tu.ID, "讲义.docx", "勾股定理：a²+b²=c²")
	require.NoError(t, err)
	require.Equal(t, 13, m.Chars, "counted in characters, not bytes")
	list, err := repo.List(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 13, list[0].MaterialChars)
	mats, _ := repo.ListMaterials(ctx, tu.ID, false)
	require.Empty(t, mats[0].Content)
	mats, _ = repo.ListMaterials(ctx, tu.ID, true)
	require.Equal(t, "勾股定理：a²+b²=c²", mats[0].Content)

	s1, err := repo.UpsertStudent(ctx, tu.ID, "小明", "h1")
	require.NoError(t, err)
	s2, err := repo.UpsertStudent(ctx, tu.ID, "小明", "h2")
	require.NoError(t, err)
	require.Equal(t, s1.ID, s2.ID, "same name, same student")
	gone, _ := repo.StudentByToken(ctx, tu.ID, "h1")
	require.Nil(t, gone, "the old token is replaced")
	cur, _ := repo.StudentByToken(ctx, tu.ID, "h2")
	require.Equal(t, s1.ID, cur.ID)

	require.NoError(t, repo.AddMessage(ctx, tu.ID, s1.ID, "什么是勾股定理", "直角三角形……"))
	require.NoError(t, repo.AddMessage(ctx, tu.ID, s1.ID, "举个例子", "3、4、5"))
	msgs, err := repo.ListMessages(ctx, tu.ID, time.Now().Add(-time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, "举个例子", msgs[0].Question)
	require.Equal(t, "小明", msgs[0].StudentName)
	n, _ := repo.CountMessages(ctx, tu.ID, time.Now().Add(-time.Hour))
	require.Equal(t, 2, n)
	students, _ := repo.ListStudents(ctx, tu.ID)
	require.Equal(t, 2, students[0].Questions)

	require.NoError(t, repo.DeleteMaterial(ctx, tu.ID, m.ID))
	require.NoError(t, repo.Delete(ctx, user.ID, tu.ID))
	n, _ = repo.CountMessages(ctx, tu.ID, time.Time{})
	require.Zero(t, n, "messages go with the assistant")
}
