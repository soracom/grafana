package api

import (
	"errors"
	"net/http"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/plugins"
	"github.com/grafana/grafana/pkg/services/plugindashboards"
	"github.com/grafana/grafana/pkg/web"
)

// GetPluginDashboards get plugin dashboards.
//
// /api/plugins/:pluginId/dashboards
func (hs *HTTPServer) GetPluginDashboards(c *models.ReqContext) response.Response {
	pluginID := web.Params(c.Req)[":pluginId"]

	listReq := &plugindashboards.ListPluginDashboardsRequest{
		OrgID:    c.OrgID,
		PluginID: pluginID,
	}
	list, err := hs.pluginDashboardService.ListPluginDashboards(c.Req.Context(), listReq)
	if err != nil {
		var notFound plugins.NotFoundError
		if errors.As(err, &notFound) {
			return response.Error(http.StatusNotFound, notFound.Error(), nil)
		}

		return response.Error(http.StatusInternalServerError, "Failed to get plugin dashboards", err)
	}

	return response.JSON(http.StatusOK, list.Items)
}
