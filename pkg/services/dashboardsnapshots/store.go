package dashboardsnapshots

import (
	"context"
)

type Store interface {
	CheckDashboardSnapshotUpdateRequired(context.Context, *CheckDashboardSnapshotUpdateRequiredCommand) error
	CreateDashboardSnapshot(context.Context, *CreateDashboardSnapshotCommand) error
	DeleteDashboardSnapshot(context.Context, *DeleteDashboardSnapshotCommand) error
	DeleteExpiredSnapshots(context.Context, *DeleteExpiredSnapshotsCommand) error
	GetDashboardSnapshot(context.Context, *GetDashboardSnapshotQuery) error
	SearchDashboardSnapshots(context.Context, *GetDashboardSnapshotsQuery) error
}
