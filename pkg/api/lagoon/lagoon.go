package lagoon

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore"
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
func GetPlan(sqlstore sqlstore.Store, ctx context.Context, orgID int64) Plan {
	query := models.GetOrgByIdQuery{Id: orgID}
	err := sqlstore.GetOrgById(ctx, &query)

	if err != nil {
		lagoonLogger.Warn("Failed to get Org ", "orgID", orgID, "error", err)
		return PlanFree
	}

	org := query.Result

	if strings.HasSuffix(org.Name, "-"+string(PlanFree)) {
		return PlanFree
	}
	if strings.HasSuffix(org.Name, "-"+string(PlanPro)) {
		return PlanPro
	}
	return PlanMaker
}

// AlertFrequency returns the fastest possible refresh rate for alert evaluation
func AlertFrequency(sqlstore sqlstore.Store, ctx context.Context, orgID int64) int64 {
	plan := GetPlan(sqlstore, ctx, orgID)

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

func HashWithOrgAccessKey(sqlstore sqlstore.Store, ctx context.Context, orgID int64, hashme string) (string, error) {

	ak, err := GetOrgAccessKey(sqlstore, ctx, orgID)

	if err != nil {
		return "", err
	}

	ak = hashme + ak
	hbytes := []byte(ak)
	return fmt.Sprintf("%x", md5.Sum(hbytes)), nil

}

// This returns the first harvest access key it finds for the Org
func GetOrgAccessKey(sqlstore sqlstore.Store, ctx context.Context, orgID int64) (string, error) {

	query := models.GetDataSourcesQuery{OrgId: orgID, DataSourceLimit: 5}
	err := sqlstore.GetDataSources(ctx, &query)

	if err != nil {
		return "", err
	}

	for _, ds := range query.Result {

		if (ds.Type == "soracom-harvest-datasource" || ds.Type == "harvest-backend-datasource") && len(ds.User) > 0 {
			return ds.User, nil
		}

	}

	return "", errors.New("could not find access key")
}

func TriggerLiveSnapshotIfNecessary(sqlstore sqlstore.Store, ctx context.Context, snapshot *models.DashboardSnapshot) (int64, error) {
	// if this is a live snapshot and it hasnt been updated for a while, put that in motion
	if strings.HasSuffix(snapshot.Key, "-live") {
		since := time.Now().Add(-1 * time.Minute)
		if snapshot.Updated.Before(since) {
			cmd := &models.CheckDashboardSnapshotUpdateRequiredCommand{Key: snapshot.Key, Since: since}
			err := sqlstore.CheckDashboardSnapshotUpdateRequired(ctx, cmd)
			if err != nil {
				return 60, err
			}
			if cmd.UpdateRequired {
				fmt.Println("********* UPDATE REQUIRED! *********")
				// extract the original url and also the org id.
				url, err := snapshot.Dashboard.Get("snapshot").Get("originalUrl").String()
				if err != nil {
					fmt.Println(err)
					return 60, errors.New("failed to extract original snapshot URL")

					//Failed to get update status for live dashboard snapshot logger=context userId=14 orgId=27 uname=soracom-admin error="failed to extract original snapshot URL" remote_addr=[::1]
				}
				go CallSnapshotRefresh(snapshot.OrgId, url)
				//cache for only 30 seconds
				return 30, nil
			}
		} else {
			fmt.Println("********* NO UPDATE REQUIRED! *********")
			// Set the cache expiry time to match when the data will need to be updated
			return int64(snapshot.Updated.Sub(since).Seconds()), nil
		}

	}

	// not a live snapshot so cache for an hour
	return 3600, nil
}

// SnapshotRequestRefresh is the data type sent to the lambda
type SnapshotRefreshRequest struct {
	URL   string `json:"url"`
	OrgID string `json:"orgId"`
}

func CallSnapshotRefresh(orgID int64, originalURL string) {

	fmt.Printf("******* CALLING SNAPSHOT REFRESH ******")
	refreshReq := SnapshotRefreshRequest{URL: originalURL, OrgID: strconv.FormatInt(orgID, 10)}

	url := os.Getenv("LAGOON_SNAPSHOT_REFRESH_URL")
	securityHeaderName := os.Getenv("LAGOON_SNAPSHOT_REFRESH_HEADER_NAME")
	securityHeaderValue := os.Getenv("LAGOON_SNAPSHOT_REFRESH_HEADER_VALUE")

	fmt.Println("Calling lambda: ", url, " with ", orgID, " to refresh:", originalURL)
	lagoonLogger.Info("Performing lagoon snapsho", "url", url)
	jsonStr, err := json.Marshal(refreshReq)

	if err != nil {
		lagoonLogger.Error("Couldn't marshall snapshot refresh request", "refreshreq", refreshReq)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

	if err != nil {
		lagoonLogger.Error("Couldn't create snapshot refresh request", "refreshreq", refreshReq)
		return
	}

	req.Header.Set(securityHeaderName, securityHeaderValue)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		lagoonLogger.Error("Snapshot refresh failed", "error", err, "orgID", orgID, "originalURL", originalURL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			lagoonLogger.Error("Error reading snapshot lambda response", "error", err, "orgID", orgID, "originalURL", originalURL)
		}
		bodyString := string(bodyBytes)
		lagoonLogger.Error("Problem calling snapshot lambda", "code", resp.Status, "responsebody", bodyString, "orgID", orgID, "originalURL", originalURL)
	}

	fmt.Println("****** SNAPSHOT REFRESH SUCCESS! ******", resp)
}
