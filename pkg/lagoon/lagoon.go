package lagoon

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/org"
)

var (
	lagoonLogger = log.New("lagoon")
)

// Plan is the Lagoon plan level
type Plan string

const (
	// PlanMaker is the standard paid plan
	PlanMaker Plan = "MAKER"
	// PlanPro is for premium users
	PlanPro Plan = "PRO"
	// PlanFree is the free level
	PlanFree Plan = "FREE"
)

// GetPlan returns the Lagoon plan the current org is on
func GetPlan(orgservice org.Service, ctx context.Context, orgID int64) Plan {
	query := org.GetOrgByIDQuery{ID: orgID}
	org, err := orgservice.GetByID(ctx, &query)

	if err != nil {
		lagoonLogger.Warn("Failed to get Org ", "orgID", orgID, "error", err)
		return PlanFree
	}

	return GetPlanFromOrgName(org.Name)

}

// GetPlanFromOrgName parses the orgname and determines through some magic code
// what plan the customer is on
func GetPlanFromOrgName(orgName string) Plan {
	if strings.HasSuffix(orgName, "-"+string(PlanFree)) {
		return PlanFree
	}
	if strings.HasSuffix(orgName, "-"+string(PlanPro)) {
		return PlanPro
	}
	return PlanMaker
}

// AlertFrequencyForOrg returns the fastest possible refresh rate for alert evaluation
func AlertFrequencyForPlan(plan Plan) int64 {

	switch plan {
	case PlanFree:
		return 60
	case PlanMaker:
		return 30
	case PlanPro:
		return 5
	}

	// Just in case something weird happened.
	return 60
}

// PublicDashboardsEnabledForPlan returns true if they are
func PublicDashboardsEnabledForPlan(plan Plan) bool {
	return plan == PlanPro
}

func IsFreePlan(orgName string) bool {
	return strings.HasSuffix(orgName, "-"+string(PlanFree))
}

func IsProPlan(orgName string) bool {
	return strings.HasSuffix(orgName, "-"+string(PlanPro))
}

func IsMakerPlan(orgName string) bool {
	if IsProPlan(orgName) || IsFreePlan(orgName) {
		return false
	}

	//if it isn't an empty string then it is good
	return len(orgName) > 0
}

func HashWithOrgAccessKey(dsService datasources.DataSourceService, ctx context.Context, orgID int64, hashme string) (string, error) {

	ak, err := GetOrgAccessKey(dsService, ctx, orgID)

	if err != nil {
		return "", err
	}

	ak = hashme + ak
	hbytes := []byte(ak)
	return fmt.Sprintf("%x", md5.Sum(hbytes)), nil

}

// This returns the first harvest access key it finds for the Org
func GetOrgAccessKey(dsService datasources.DataSourceService, ctx context.Context, orgID int64) (string, error) {
	query := datasources.GetDataSourcesQuery{OrgID: orgID, DataSourceLimit: 5}

	dsList, err := dsService.GetDataSources(ctx, &query)
	if err != nil {
		return "", err
	}

	for _, ds := range dsList {

		if (ds.Type == "soracom-harvest-datasource" || ds.Type == "harvest-backend-datasource") && len(ds.User) > 0 {
			return ds.User, nil
		}

	}

	return "", errors.New("could not find access key")
}
