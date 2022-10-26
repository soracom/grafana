package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/services/dashboardsnapshots"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/setting"
)

type DashboardSnapshotStore struct {
	store db.DB
	log   log.Logger
}

// DashboardStore implements the Store interface
var _ dashboardsnapshots.Store = (*DashboardSnapshotStore)(nil)

func ProvideStore(db db.DB) *DashboardSnapshotStore {
	return &DashboardSnapshotStore{store: db, log: log.New("dashboardsnapshot.store")}
}

// DeleteExpiredSnapshots removes snapshots with old expiry dates.
// SnapShotRemoveExpired is deprecated and should be removed in the future.
// Snapshot expiry is decided by the user when they share the snapshot.
func (d *DashboardSnapshotStore) DeleteExpiredSnapshots(ctx context.Context, cmd *dashboardsnapshots.DeleteExpiredSnapshotsCommand) error {
	return d.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		if !setting.SnapShotRemoveExpired {
			d.log.Warn("[Deprecated] The snapshot_remove_expired setting is outdated. Please remove from your config.")
			return nil
		}

		deleteExpiredSQL := "DELETE FROM dashboard_snapshot WHERE expires < ?"
		expiredResponse, err := sess.Exec(deleteExpiredSQL, time.Now())
		if err != nil {
			return err
		}
		cmd.DeletedRows, _ = expiredResponse.RowsAffected()

		return nil
	})
}

func (d *DashboardSnapshotStore) CreateDashboardSnapshot(ctx context.Context, cmd *dashboardsnapshots.CreateDashboardSnapshotCommand) error {
	return d.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		var expires = time.Now().Add(time.Hour * 24 * 365 * 50)
		if cmd.Expires > 0 {
			expires = time.Now().Add(time.Second * time.Duration(cmd.Expires))
		}

		snapshot := &dashboardsnapshots.DashboardSnapshot{
			Name:               cmd.Name,
			Key:                cmd.Key,
			DeleteKey:          cmd.DeleteKey,
			OrgId:              cmd.OrgId,
			UserId:             cmd.UserId,
			External:           cmd.External,
			ExternalUrl:        cmd.ExternalUrl,
			ExternalDeleteUrl:  cmd.ExternalDeleteUrl,
			Dashboard:          simplejson.New(),
			DashboardEncrypted: cmd.DashboardEncrypted,
			Expires:            expires,
			Created:            time.Now(),
			Updated:            time.Now(),
		}

		// Check to see if snapshot already exists, if it does update
		var err error
		query := &dashboardsnapshots.GetDashboardSnapshotQuery{DeleteKey: cmd.DeleteKey}
		if strings.HasSuffix(cmd.DeleteKey, "-live") && d.GetDashboardSnapshot(ctx, query) == nil {
			cmd.Key = query.Result.Key
			snapshot.Key = cmd.Key
			snapshot.UserId = query.Result.UserId // we want to maintain the user id that created the original snapshot
			fmt.Println("****UPDATING with key:", cmd.Key, " UserID:", query.Result.UserId)
			cond := &dashboardsnapshots.DashboardSnapshot{
				Key:   cmd.Key,
				OrgId: cmd.OrgId,
			}
			_, err = sess.Update(snapshot, cond)
		} else {
			_, err = sess.Insert(snapshot)
		}
		cmd.Result = snapshot

		return err
	})
}

func (d *DashboardSnapshotStore) DeleteDashboardSnapshot(ctx context.Context, cmd *dashboardsnapshots.DeleteDashboardSnapshotCommand) error {
	return d.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		var rawSQL = "DELETE FROM dashboard_snapshot WHERE delete_key=?"
		_, err := sess.Exec(rawSQL, cmd.DeleteKey)
		return err
	})
}

func (d *DashboardSnapshotStore) GetDashboardSnapshot(ctx context.Context, query *dashboardsnapshots.GetDashboardSnapshotQuery) error {
	return d.store.WithDbSession(ctx, func(sess *db.Session) error {
		snapshot := dashboardsnapshots.DashboardSnapshot{Key: query.Key, DeleteKey: query.DeleteKey}
		has, err := sess.Get(&snapshot)

		if err != nil {
			return err
		} else if !has {
			return dashboardsnapshots.ErrBaseNotFound.Errorf("dashboard snapshot not found")
		}

		query.Result = &snapshot
		return nil
	})
}

// CheckDashboardSnapshotUpdateRequired looks to see if a live snapshot has been updated
// since the specified time.  If it hasn't it sets the updated time to Now() and returns
// true so a one-time update script/lambda/external call can be triggered by the caller
func (d *DashboardSnapshotStore) CheckDashboardSnapshotUpdateRequired(ctx context.Context, cmd *dashboardsnapshots.CheckDashboardSnapshotUpdateRequiredCommand) error {

	return d.store.WithTransactionalDbSession(ctx, func(sess *db.Session) error {
		query := &dashboardsnapshots.GetDashboardSnapshotQuery{Key: cmd.Key}

		err := d.GetDashboardSnapshot(ctx, query)

		if err != nil {
			return err
		}

		snapshot := query.Result

		if query.Result.Updated.After(cmd.Since) {
			// update is already being processed
			// this could happen if another request came in at a similar time
			return nil
		}

		cmd.UpdateRequired = true

		// reset the update time to now so we dont update again before time
		snapshot.Updated = time.Now()

		cond := &dashboardsnapshots.DashboardSnapshot{
			Key: cmd.Key,
		}
		_, err = sess.Update(snapshot, cond)

		return err
	})
}

// SearchDashboardSnapshots returns a list of all snapshots for admins
// for other roles, it returns snapshots created by the user
func (d *DashboardSnapshotStore) SearchDashboardSnapshots(ctx context.Context, query *dashboardsnapshots.GetDashboardSnapshotsQuery) error {
	return d.store.WithDbSession(ctx, func(sess *db.Session) error {
		var snapshots = make(dashboardsnapshots.DashboardSnapshotsList, 0)
		if query.Limit > 0 {
			sess.Limit(query.Limit)
		}
		sess.Table("dashboard_snapshot")

		if query.Name != "" {
			sess.Where("name LIKE ?", query.Name)
		}

		// admins can see all snapshots, everyone else can only see their own snapshots
		switch {
		case query.SignedInUser.OrgRole == org.RoleAdmin:
			sess.Where("org_id = ?", query.OrgId)
		case !query.SignedInUser.IsAnonymous:
			sess.Where("org_id = ? AND user_id = ?", query.OrgId, query.SignedInUser.UserID)
		default:
			query.Result = snapshots
			return nil
		}

		err := sess.Find(&snapshots)
		query.Result = snapshots
		return err
	})
}
