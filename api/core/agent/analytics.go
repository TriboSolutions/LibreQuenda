package agent

import (
	"context"
	"errors"

	"librequenda/core/analytics"
	"librequenda/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The agent itself writes the SQL here (no language model of the installation's):
// the same read-only user, views and limits as the indicators' assistant, for the
// councils the signed-in person administers, or all of them for a superadministrator.

var errNotAdmin = errors.New("only council administrators and superadministrators look at the analytics, and only for the councils they run")

// scope: what the signed-in person may analyse.
func (a *agent) scope(ctx context.Context, req *mcp.CallToolRequest) (analytics.Scope, error) {
	var userId, err = a.user(ctx, req)
	if err != nil {
		return analytics.Scope{}, err
	}
	var queries = model.New(a.db)
	if super, superErr := queries.IsPlatformAdmin(ctx, userId); superErr == nil && super {
		return analytics.Scope{Everything: true}, nil
	}
	var roles, rolesErr = queries.GetUserTenantRoles(ctx, userId)
	if rolesErr != nil {
		return analytics.Scope{}, errors.New("your councils could not be read")
	}
	var scope analytics.Scope
	for _, role := range roles {
		if role.Role == model.TenantRoleAdmin {
			scope.Councils = append(scope.Councils, role.TenantID)
		}
	}
	if len(scope.Councils) == 0 {
		return scope, errNotAdmin
	}
	return scope, nil
}

type ViewsOut struct {
	Views []analytics.View `json:"views"`
	Note  string           `json:"note"`
}

func (a *agent) analyticsViews(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ViewsOut, error) {
	if _, err := a.scope(ctx, req); err != nil {
		return nil, ViewsOut{}, err
	}
	var views, err = analytics.Views(ctx)
	if err != nil {
		return nil, ViewsOut{}, err
	}
	return nil, ViewsOut{Views: views, Note: "PostgreSQL views (schema analytics). Timestamps are timestamptz: use AT TIME ZONE 'Europe/Madrid' for local days and hours. People appear by id only; aggregate."}, nil
}

type QueryIn struct {
	Sql string `json:"sql" jsonschema:"one SELECT (or WITH … SELECT) on the analytics views; at most 500 rows come back"`
}

func (a *agent) analyticsQuery(ctx context.Context, req *mcp.CallToolRequest, in QueryIn) (*mcp.CallToolResult, analytics.Result, error) {
	var scope, err = a.scope(ctx, req)
	if err != nil {
		return nil, analytics.Result{}, err
	}
	var result, runErr = analytics.Run(ctx, scope, in.Sql)
	if runErr != nil {
		return nil, analytics.Result{}, runErr
	}
	return nil, *result, nil
}
