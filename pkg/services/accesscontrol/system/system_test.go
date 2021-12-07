package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/accesscontrol/database"
	accesscontrolmock "github.com/grafana/grafana/pkg/services/accesscontrol/mock"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/web"
)

type getDescriptionTestCase struct {
	desc           string
	options        Options
	permissions    []*accesscontrol.Permission
	expected       Description
	expectedStatus int
}

func TestSystem_getDescription(t *testing.T) {
	tests := []getDescriptionTestCase{
		{
			desc: "should return description",
			options: Options{
				Resource: "dashboards",
				Assignments: Assignments{
					Users:        true,
					Teams:        true,
					BuiltInRoles: true,
				},
				PermissionsToActions: map[string][]string{
					"View":  {"dashboards:read"},
					"Edit":  {"dashboards:read", "dashboards:write", "dashboards:delete"},
					"Admin": {"dashboards:read", "dashboards:write", "dashboards:delete", "dashboards.permissions:read", "dashboards:permissions:write"},
				},
			},
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read"},
			},
			expected: Description{
				Assignments: Assignments{
					Users:        true,
					Teams:        true,
					BuiltInRoles: true,
				},
				Permissions: []string{"Admin", "Edit", "View"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			desc: "should only return user assignment",
			options: Options{
				Resource: "dashboards",
				Assignments: Assignments{
					Users:        true,
					Teams:        false,
					BuiltInRoles: false,
				},
				PermissionsToActions: map[string][]string{
					"View": {"dashboards:read"},
				},
			},
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read"},
			},
			expected: Description{
				Assignments: Assignments{
					Users:        true,
					Teams:        false,
					BuiltInRoles: false,
				},
				Permissions: []string{"View"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			desc: "should return 403 when missing read permission",
			options: Options{
				Resource: "dashboards",
				Assignments: Assignments{
					Users:        true,
					Teams:        false,
					BuiltInRoles: false,
				},
				PermissionsToActions: map[string][]string{
					"View": {"dashboards:read"},
				},
			},
			permissions:    []*accesscontrol.Permission{},
			expected:       Description{},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, server, _ := setupTestEnvironment(t, &models.SignedInUser{}, tt.permissions, tt.options)
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/access-control/system/%s/description", tt.options.Resource), nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)

			got := Description{}
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&got))
			assert.Equal(t, tt.expected, got)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedStatus, recorder.Code)
			}
		})
	}
}

type getPermissionsTestCase struct {
	desc           string
	resourceID     string
	permissions    []*accesscontrol.Permission
	expectedStatus int
}

func TestSystem_getPermissions(t *testing.T) {
	tests := []getPermissionsTestCase{
		{
			desc:           "expect permissions for resource with id 1",
			resourceID:     "1",
			permissions:    []*accesscontrol.Permission{{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"}},
			expectedStatus: 200,
		},
		{
			desc:           "expect http status 403 when missing permission",
			resourceID:     "1",
			permissions:    []*accesscontrol.Permission{},
			expectedStatus: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			system, server, sql := setupTestEnvironment(t, &models.SignedInUser{OrgId: 1}, tt.permissions, testOptions)

			// seed team 1 with "Edit" permission on dashboard 1
			team, err := sql.CreateTeam("test", "test@test.com", 1)
			require.NoError(t, err)
			_, err = system.SetTeamPermission(context.Background(), team.OrgId, team.Id, tt.resourceID, []string{"dashboards:read", "dashboards:write", "dashboards:delete"})
			require.NoError(t, err)
			// seed user 1 with "View" permission on dashboard 1
			u, err := sql.CreateUser(context.Background(), models.CreateUserCommand{Login: "test", OrgId: 1})
			require.NoError(t, err)
			_, err = system.SetUserPermission(context.Background(), u.OrgId, u.Id, tt.resourceID, []string{"dashboards:read"})
			require.NoError(t, err)

			// seed built in role Admin with "View" permission on dashboard 1
			_, err = system.SetBuiltinRolePermission(context.Background(), 1, "Admin", tt.resourceID, []string{"dashboards:read", "dashboards:write", "dashboards:delete"})
			require.NoError(t, err)

			permissions, recorder := getPermission(t, server, testOptions.Resource, tt.resourceID)
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus == http.StatusOK {
				for _, p := range permissions {
					if p.UserID != 0 {
						assert.Equal(t, "View", p.Permission)
					} else if p.TeamID != 0 {
						assert.Equal(t, "Edit", p.Permission)
					} else {
						assert.Equal(t, "Edit", p.Permission)
					}
				}
			}
		})
	}
}

type setBuiltinPermissionTestCase struct {
	desc           string
	resourceID     string
	builtInRole    string
	expectedStatus int
	permission     string
	permissions    []*accesscontrol.Permission
}

func TestSystem_setBuiltinRolePermission(t *testing.T) {
	tests := []setBuiltinPermissionTestCase{
		{
			desc:           "should set Edit permission for Viewer",
			resourceID:     "1",
			builtInRole:    "Viewer",
			expectedStatus: 200,
			permission:     "Edit",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set View permission for Admin",
			resourceID:     "1",
			builtInRole:    "Admin",
			expectedStatus: 200,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set return http 400 for invalid built in role",
			resourceID:     "1",
			builtInRole:    "Invalid",
			expectedStatus: http.StatusBadRequest,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set return http 403 when missing permissions",
			resourceID:     "1",
			builtInRole:    "Invalid",
			expectedStatus: http.StatusForbidden,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, server, _ := setupTestEnvironment(t, &models.SignedInUser{OrgId: 1}, tt.permissions, testOptions)

			recorder := setPermission(t, server, testOptions.Resource, tt.resourceID, tt.permission, "builtInRoles", tt.builtInRole)
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus == http.StatusOK {
				permissions, _ := getPermission(t, server, testOptions.Resource, tt.resourceID)
				require.Len(t, permissions, 1)
				assert.Equal(t, tt.permission, permissions[0].Permission)
				assert.Equal(t, tt.builtInRole, permissions[0].BuiltInRole)
			}
		})
	}
}

type setTeamPermissionTestCase struct {
	desc           string
	teamID         int64
	resourceID     string
	expectedStatus int
	permission     string
	permissions    []*accesscontrol.Permission
}

func TestSystem_setTeamPermission(t *testing.T) {
	tests := []setTeamPermissionTestCase{
		{
			desc:           "should set Edit permission for team 1",
			teamID:         1,
			resourceID:     "1",
			expectedStatus: 200,
			permission:     "Edit",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set View permission for team 1",
			teamID:         1,
			resourceID:     "1",
			expectedStatus: 200,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set return http 400 when team does not exist",
			teamID:         2,
			resourceID:     "1",
			expectedStatus: http.StatusBadRequest,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set return http 403 when missing permissions",
			teamID:         2,
			resourceID:     "1",
			expectedStatus: http.StatusForbidden,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, server, sql := setupTestEnvironment(t, &models.SignedInUser{OrgId: 1}, tt.permissions, testOptions)

			// seed team
			_, err := sql.CreateTeam("test", "test@test.com", 1)
			require.NoError(t, err)

			recorder := setPermission(t, server, testOptions.Resource, tt.resourceID, tt.permission, "teams", strconv.Itoa(int(tt.teamID)))
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			if tt.expectedStatus == http.StatusOK {
				permissions, _ := getPermission(t, server, testOptions.Resource, tt.resourceID)
				require.Len(t, permissions, 1)
				assert.Equal(t, tt.permission, permissions[0].Permission)
				assert.Equal(t, tt.teamID, permissions[0].TeamID)
			}
		})
	}
}

type setUserPermissionTestCase struct {
	desc           string
	userID         int64
	resourceID     string
	expectedStatus int
	permission     string
	permissions    []*accesscontrol.Permission
}

func TestSystem_setUserPermission(t *testing.T) {
	tests := []setUserPermissionTestCase{
		{
			desc:           "should set Edit permission for user 1",
			userID:         1,
			resourceID:     "1",
			expectedStatus: 200,
			permission:     "Edit",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set View permission for user 1",
			userID:         1,
			resourceID:     "1",
			expectedStatus: 200,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set return http 400 when user does not exist",
			userID:         2,
			resourceID:     "1",
			expectedStatus: http.StatusBadRequest,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
				{Action: "dashboards.permissions:write", Scope: "dashboards:id:1"},
			},
		},
		{
			desc:           "should set return http 403 when missing permissions",
			userID:         2,
			resourceID:     "1",
			expectedStatus: http.StatusForbidden,
			permission:     "View",
			permissions: []*accesscontrol.Permission{
				{Action: "dashboards.permissions:read", Scope: "dashboards:id:1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			_, server, sql := setupTestEnvironment(t, &models.SignedInUser{OrgId: 1}, tt.permissions, testOptions)

			// seed team
			_, err := sql.CreateUser(context.Background(), models.CreateUserCommand{Login: "test", OrgId: 1})
			require.NoError(t, err)

			recorder := setPermission(t, server, testOptions.Resource, tt.resourceID, tt.permission, "users", strconv.Itoa(int(tt.userID)))
			assert.Equal(t, tt.expectedStatus, recorder.Code)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			if tt.expectedStatus == http.StatusOK {
				permissions, _ := getPermission(t, server, testOptions.Resource, tt.resourceID)
				require.Len(t, permissions, 1)
				assert.Equal(t, tt.permission, permissions[0].Permission)
				assert.Equal(t, tt.userID, permissions[0].UserID)
			}
		})
	}
}

func setupTestEnvironment(t *testing.T, user *models.SignedInUser, permissions []*accesscontrol.Permission, ops Options) (*System, *web.Mux, *sqlstore.SQLStore) {
	sql := sqlstore.InitTestDB(t)
	store := database.ProvideService(sql)

	system, err := New(ops, routing.NewRouteRegister(), accesscontrolmock.New().WithPermissions(permissions), store)
	require.NoError(t, err)

	server := web.New()
	server.UseMiddleware(web.Renderer(path.Join(setting.StaticRootPath, "views"), "[[", "]]"))
	server.Use(contextProvider(&testContext{user}))
	system.api.router.Register(server)

	return system, server, sql
}

type testContext struct {
	user *models.SignedInUser
}

func contextProvider(tc *testContext) web.Handler {
	return func(c *web.Context) {
		signedIn := tc.user != nil
		reqCtx := &models.ReqContext{
			Context:      c,
			SignedInUser: tc.user,
			IsSignedIn:   signedIn,
			SkipCache:    true,
			Logger:       log.New("test"),
		}
		c.Map(reqCtx)
	}
}

var testOptions = Options{
	Resource: "dashboards",
	Assignments: Assignments{
		Users:        true,
		Teams:        true,
		BuiltInRoles: true,
	},
	PermissionsToActions: map[string][]string{
		"View": {"dashboards:read"},
		"Edit": {"dashboards:read", "dashboards:write", "dashboards:delete"},
	},
}

func getPermission(t *testing.T, server *web.Mux, resource, resourceID string) ([]resourcePermissionDTO, *httptest.ResponseRecorder) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/access-control/system/%s/%s", resource, resourceID), nil)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	var permissions []resourcePermissionDTO
	if recorder.Code == http.StatusOK {
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&permissions))
	}
	return permissions, recorder
}

func setPermission(t *testing.T, server *web.Mux, resource, resourceID, permission, assignment, assignTo string) *httptest.ResponseRecorder {
	body := strings.NewReader(fmt.Sprintf(`{"permission": "%s"}`, permission))
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/access-control/system/%s/%s/%s/%s", resource, resourceID, assignment, assignTo), body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)

	return recorder
}
