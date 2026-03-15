package web

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/rapha/clockkeeper/ent"
	"github.com/rapha/clockkeeper/ent/user"
	clockkeeperv1 "github.com/rapha/clockkeeper/gen/clockkeeper/v1"
)

// ClockKeeperServiceHandler implements the ConnectRPC ClockKeeperService.
type ClockKeeperServiceHandler struct {
	config *Config
	db     *ent.Client
	auth   *AuthInterceptor
}

func (h *ClockKeeperServiceHandler) Login(ctx context.Context, req *connect.Request[clockkeeperv1.LoginRequest]) (*connect.Response[clockkeeperv1.LoginResponse], error) {
	u, err := h.db.User.Query().
		Where(user.Username(req.Msg.Username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !CheckPassword(req.Msg.Password, u.PasswordHash) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}

	token, err := h.auth.IssueToken(req.Msg.Username)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&clockkeeperv1.LoginResponse{Token: token}), nil
}
