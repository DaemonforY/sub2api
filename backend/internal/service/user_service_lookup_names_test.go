//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsernamesByPlaceholderEmails(t *testing.T) {
	const wechat = "wechat-o0pO06n89o@wechat-connect.invalid"
	repo := &userRepoStub{usersByEmail: map[string]*User{
		wechat:                                 {ID: 84, Email: wechat, Username: "波西米亚格"},
		"wechat-noname@wechat-connect.invalid": {ID: 85, Email: "wechat-noname@wechat-connect.invalid", Username: "  "},
		"alice@example.com":                    {ID: 1, Email: "alice@example.com", Username: "alice"},
	}}
	svc := NewUserService(repo, nil, nil, nil)

	names, err := svc.UsernamesByPlaceholderEmails(context.Background(), []string{
		wechat, " " + wechat + " ", "wechat-noname@wechat-connect.invalid",
		"wechat-unknown@wechat-connect.invalid", "alice@example.com", "not-an-email.invalid",
	})
	require.NoError(t, err)
	// Real addresses are never looked up; unknown users and blank usernames are left out.
	require.Equal(t, map[string]string{wechat: "波西米亚格"}, names)
}

func TestUsernamesByPlaceholderEmailsPropagatesRepoErrors(t *testing.T) {
	svc := NewUserService(&userRepoStub{getByEmailErr: errors.New("db down")}, nil, nil, nil)
	_, err := svc.UsernamesByPlaceholderEmails(context.Background(), []string{"wechat-x@wechat-connect.invalid"})
	require.Error(t, err)
}
