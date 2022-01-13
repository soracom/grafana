package api

import (
	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/util"
)

// GetOrgDashboards GET /api/orgs/:orgId/Dashboards
func GetOrgDashboards(c *models.ReqContext) response.Response {
	return getOrgDashboardsHelper(c, c.ParamsInt64(":orgId"))
}

func getOrgDashboardsHelper(c *models.ReqContext, orgID int64) response.Response {
	query := models.GetDashboardsByOrgIdQuery{
		OrgId: orgID,
	}

	if err := bus.DispatchCtx(c.Req.Context(), &query); err != nil {
		return response.Error(500, "Failed to get dashboards", err)
	}

	return response.JSON(200, query.Result)
}

// GetOrgDashboard GET /api/orgs/:orgId/dashboards/:dashId
func GetOrgDashboard(c *models.ReqContext) response.Response {
	return getOrgDashboardHelper(c, c.ParamsInt64(":orgId"), c.ParamsInt64(":dashId"))
}

func getOrgDashboardHelper(c *models.ReqContext, orgID, dashID int64) response.Response {

	q := models.GetDashboardQuery{OrgId: orgID, Id: dashID}

	if err := bus.DispatchCtx(c.Req.Context(), &q); err != nil {
		return response.Error(500, "Failed to get dashboards", err)
	}

	return response.JSON(200, q.Result)
}

// DeleteOrgDashboard DELETE /api/orgs/:orgId/dashboards/:dashId
func DeleteOrgDashboard(c *models.ReqContext) response.Response {
	return deleteOrgDashboardHelper(c, c.ParamsInt64(":orgId"), c.ParamsInt64(":dashId"))
}

func deleteOrgDashboardHelper(c *models.ReqContext, orgID, dashID int64) response.Response {
	cmd := models.DeleteDashboardCommand{OrgId: orgID, Id: dashID}
	if err := bus.DispatchCtx(c.Req.Context(), &cmd); err != nil {
		return response.Error(500, "Failed to delete dashboard", err)
	}

	return response.JSON(200, util.DynMap{
		"message": "Deleted",
	})
}
