package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/middleware"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/serviceaccounts"
	"github.com/grafana/grafana/pkg/services/serviceaccounts/database"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

type ServiceAccountsAPI struct {
	cfg            *setting.Cfg
	service        serviceaccounts.Service
	accesscontrol  accesscontrol.AccessControl
	RouterRegister routing.RouteRegister
	store          serviceaccounts.Store
	log            log.Logger
}

func NewServiceAccountsAPI(
	cfg *setting.Cfg,
	service serviceaccounts.Service,
	accesscontrol accesscontrol.AccessControl,
	routerRegister routing.RouteRegister,
	store serviceaccounts.Store,
) *ServiceAccountsAPI {
	return &ServiceAccountsAPI{
		cfg:            cfg,
		service:        service,
		accesscontrol:  accesscontrol,
		RouterRegister: routerRegister,
		store:          store,
		log:            log.New("serviceaccounts.api"),
	}
}

func (api *ServiceAccountsAPI) RegisterAPIEndpoints(
	features featuremgmt.FeatureToggles,
) {
	if !features.IsEnabled(featuremgmt.FlagServiceAccounts) {
		return
	}

	auth := accesscontrol.Middleware(api.accesscontrol)
	api.RouterRegister.Group("/api/serviceaccounts", func(serviceAccountsRoute routing.RouteRegister) {
		serviceAccountsRoute.Get("/search", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionRead)), routing.Wrap(api.SearchOrgServiceAccountsWithPaging))
		serviceAccountsRoute.Post("/", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionCreate)), routing.Wrap(api.CreateServiceAccount))
		serviceAccountsRoute.Get("/:serviceAccountId", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionRead, serviceaccounts.ScopeID)), routing.Wrap(api.RetrieveServiceAccount))
		serviceAccountsRoute.Patch("/:serviceAccountId", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionWrite, serviceaccounts.ScopeID)), routing.Wrap(api.UpdateServiceAccount))
		serviceAccountsRoute.Delete("/:serviceAccountId", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionDelete, serviceaccounts.ScopeID)), routing.Wrap(api.DeleteServiceAccount))
		serviceAccountsRoute.Get("/:serviceAccountId/tokens", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionRead, serviceaccounts.ScopeID)), routing.Wrap(api.ListTokens))
		serviceAccountsRoute.Post("/:serviceAccountId/tokens", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionWrite, serviceaccounts.ScopeID)), routing.Wrap(api.CreateToken))
		serviceAccountsRoute.Delete("/:serviceAccountId/tokens/:tokenId", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionWrite, serviceaccounts.ScopeID)), routing.Wrap(api.DeleteToken))
		serviceAccountsRoute.Get("/migrationstatus", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionRead)), routing.Wrap(api.GetAPIKeysMigrationStatus))
		serviceAccountsRoute.Post("/hideApiKeys", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionCreate)), routing.Wrap(api.HideApiKeysTab))
		serviceAccountsRoute.Post("/migrate", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionCreate)), routing.Wrap(api.MigrateApiKeysToServiceAccounts))
		serviceAccountsRoute.Post("/migrate/:keyId", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionCreate)), routing.Wrap(api.ConvertToServiceAccount))
		serviceAccountsRoute.Post("/revert/:keyId", auth(middleware.ReqOrgAdmin,
			accesscontrol.EvalPermission(serviceaccounts.ActionDelete)), routing.Wrap(api.RevertApiKey))
	})
}

// POST /api/serviceaccounts
func (api *ServiceAccountsAPI) CreateServiceAccount(c *models.ReqContext) response.Response {
	type createServiceAccountForm struct {
		Name string `json:"name" binding:"Required"`
	}
	cmd := createServiceAccountForm{}
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "Bad request data", err)
	}

	serviceAccount, err := api.store.CreateServiceAccount(c.Req.Context(), c.OrgId, cmd.Name)
	switch {
	case errors.Is(err, &database.ErrSAInvalidName{}):
		return response.Error(http.StatusBadRequest, "Failed due to %s", err)
	case err != nil:
		return response.Error(http.StatusInternalServerError, "Failed to create service account", err)
	}

	return response.JSON(http.StatusCreated, serviceAccount)
}

// GET /api/serviceaccounts/:serviceAccountId
func (api *ServiceAccountsAPI) RetrieveServiceAccount(ctx *models.ReqContext) response.Response {
	scopeID, err := strconv.ParseInt(web.Params(ctx.Req)[":serviceAccountId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Service Account ID is invalid", err)
	}

	serviceAccount, err := api.store.RetrieveServiceAccount(ctx.Req.Context(), ctx.OrgId, scopeID)
	if err != nil {
		switch {
		case errors.Is(err, serviceaccounts.ErrServiceAccountNotFound):
			return response.Error(http.StatusNotFound, "Failed to retrieve service account", err)
		default:
			return response.Error(http.StatusInternalServerError, "Failed to retrieve service account", err)
		}
	}

	saIDString := strconv.FormatInt(serviceAccount.Id, 10)
	metadata := api.getAccessControlMetadata(ctx, map[string]bool{saIDString: true})
	serviceAccount.AvatarUrl = dtos.GetGravatarUrlWithDefault("", serviceAccount.Name)
	serviceAccount.AccessControl = metadata[saIDString]

	tokens, err := api.store.ListTokens(ctx.Req.Context(), serviceAccount.OrgId, serviceAccount.Id)
	if err != nil {
		api.log.Warn("Failed to list tokens for service account", "serviceAccount", serviceAccount.Id)
	}
	serviceAccount.Tokens = int64(len(tokens))

	return response.JSON(http.StatusOK, serviceAccount)
}

// PATCH /api/serviceaccounts/:serviceAccountId
func (api *ServiceAccountsAPI) UpdateServiceAccount(c *models.ReqContext) response.Response {
	scopeID, err := strconv.ParseInt(web.Params(c.Req)[":serviceAccountId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Service Account ID is invalid", err)
	}

	var cmd serviceaccounts.UpdateServiceAccountForm
	if err := web.Bind(c.Req, &cmd); err != nil {
		return response.Error(http.StatusBadRequest, "Bad request data", err)
	}

	if cmd.Role != nil && !cmd.Role.IsValid() {
		return response.Error(http.StatusBadRequest, "Invalid role specified", nil)
	}
	if cmd.Role != nil && !c.OrgRole.Includes(*cmd.Role) {
		return response.Error(http.StatusForbidden, "Cannot assign a role higher than user's role", nil)
	}

	resp, err := api.store.UpdateServiceAccount(c.Req.Context(), c.OrgId, scopeID, &cmd)
	if err != nil {
		switch {
		case errors.Is(err, serviceaccounts.ErrServiceAccountNotFound):
			return response.Error(http.StatusNotFound, "Failed to retrieve service account", err)
		default:
			return response.Error(http.StatusInternalServerError, "Failed update service account", err)
		}
	}

	saIDString := strconv.FormatInt(resp.Id, 10)
	metadata := api.getAccessControlMetadata(c, map[string]bool{saIDString: true})
	resp.AvatarUrl = dtos.GetGravatarUrlWithDefault("", resp.Name)
	resp.AccessControl = metadata[saIDString]

	return response.JSON(http.StatusOK, util.DynMap{
		"message":        "Service account updated",
		"id":             resp.Id,
		"name":           resp.Name,
		"serviceaccount": resp,
	})
}

// DELETE /api/serviceaccounts/:serviceAccountId
func (api *ServiceAccountsAPI) DeleteServiceAccount(ctx *models.ReqContext) response.Response {
	scopeID, err := strconv.ParseInt(web.Params(ctx.Req)[":serviceAccountId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Service account ID is invalid", err)
	}
	err = api.service.DeleteServiceAccount(ctx.Req.Context(), ctx.OrgId, scopeID)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Service account deletion error", err)
	}
	return response.Success("Service account deleted")
}

// SearchOrgServiceAccountsWithPaging is an HTTP handler to search for org users with paging.
// GET /api/serviceaccounts/search
func (api *ServiceAccountsAPI) SearchOrgServiceAccountsWithPaging(c *models.ReqContext) response.Response {
	ctx := c.Req.Context()
	perPage := c.QueryInt("perpage")
	if perPage <= 0 {
		perPage = 1000
	}
	page := c.QueryInt("page")
	if page < 1 {
		page = 1
	}
	// its okay that it fails, it is only filtering that might be weird, but to safe quard against any weird incoming query param
	onlyWithExpiredTokens := c.QueryBool("expiredTokens")
	onlyDisabled := c.QueryBool("disabled")
	filter := serviceaccounts.FilterIncludeAll
	if onlyWithExpiredTokens {
		filter = serviceaccounts.FilterOnlyExpiredTokens
	}
	if onlyDisabled {
		filter = serviceaccounts.FilterOnlyDisabled
	}
	serviceAccountSearch, err := api.store.SearchOrgServiceAccounts(ctx, c.OrgId, c.Query("query"), filter, page, perPage, c.SignedInUser)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Failed to get service accounts for current organization", err)
	}

	saIDs := map[string]bool{}
	for i := range serviceAccountSearch.ServiceAccounts {
		sa := serviceAccountSearch.ServiceAccounts[i]
		sa.AvatarUrl = dtos.GetGravatarUrlWithDefault("", sa.Name)

		saIDString := strconv.FormatInt(sa.Id, 10)
		saIDs[saIDString] = true
		metadata := api.getAccessControlMetadata(c, map[string]bool{saIDString: true})
		sa.AccessControl = metadata[strconv.FormatInt(sa.Id, 10)]
		tokens, err := api.store.ListTokens(ctx, sa.OrgId, sa.Id)
		if err != nil {
			api.log.Warn("Failed to list tokens for service account", "serviceAccount", sa.Id)
		}
		sa.Tokens = int64(len(tokens))
	}

	return response.JSON(http.StatusOK, serviceAccountSearch)
}

// GET /api/serviceaccounts/migrationstatus
func (api *ServiceAccountsAPI) GetAPIKeysMigrationStatus(ctx *models.ReqContext) response.Response {
	upgradeStatus, err := api.store.GetAPIKeysMigrationStatus(ctx.Req.Context(), ctx.OrgId)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "Internal server error", err)
	}
	return response.JSON(http.StatusOK, upgradeStatus)
}

// POST /api/serviceaccounts/hideapikeys
func (api *ServiceAccountsAPI) HideApiKeysTab(ctx *models.ReqContext) response.Response {
	if err := api.store.HideApiKeysTab(ctx.Req.Context(), ctx.OrgId); err != nil {
		return response.Error(http.StatusInternalServerError, "Internal server error", err)
	}
	return response.Success("API keys hidden")
}

// POST /api/serviceaccounts/migrate
func (api *ServiceAccountsAPI) MigrateApiKeysToServiceAccounts(ctx *models.ReqContext) response.Response {
	if err := api.store.MigrateApiKeysToServiceAccounts(ctx.Req.Context(), ctx.OrgId); err == nil {
		return response.Success("API keys migrated to service accounts")
	} else {
		return response.Error(http.StatusInternalServerError, "Internal server error", err)
	}
}

// POST /api/serviceaccounts/migrate/:keyId
func (api *ServiceAccountsAPI) ConvertToServiceAccount(ctx *models.ReqContext) response.Response {
	keyId, err := strconv.ParseInt(web.Params(ctx.Req)[":keyId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Key ID is invalid", err)
	}
	if err := api.store.MigrateApiKey(ctx.Req.Context(), ctx.OrgId, keyId); err == nil {
		return response.Success("Service accounts converted")
	} else {
		return response.Error(http.StatusInternalServerError, "Error converting API key", err)
	}
}

// POST /api/serviceaccounts/revert/:keyId
func (api *ServiceAccountsAPI) RevertApiKey(ctx *models.ReqContext) response.Response {
	keyId, err := strconv.ParseInt(web.Params(ctx.Req)[":keyId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "Key ID is invalid", err)
	}
	if err := api.store.RevertApiKey(ctx.Req.Context(), keyId); err != nil {
		return response.Error(http.StatusInternalServerError, "Error reverting API key", err)
	}
	return response.Success("API key reverted")
}

func (api *ServiceAccountsAPI) getAccessControlMetadata(c *models.ReqContext, saIDs map[string]bool) map[string]accesscontrol.Metadata {
	if api.accesscontrol.IsDisabled() || !c.QueryBool("accesscontrol") {
		return map[string]accesscontrol.Metadata{}
	}

	if c.SignedInUser.Permissions == nil {
		return map[string]accesscontrol.Metadata{}
	}

	permissions, ok := c.SignedInUser.Permissions[c.OrgId]
	if !ok {
		return map[string]accesscontrol.Metadata{}
	}

	return accesscontrol.GetResourcesMetadata(c.Req.Context(), permissions, "serviceaccounts:id:", saIDs)
}
