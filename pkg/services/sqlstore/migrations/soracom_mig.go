package migrations

import . "github.com/grafana/grafana/pkg/services/sqlstore/migrator"

func addSoracomDatasourceNameMigration(mg *Migrator) {
	const sql = `UPDATE data_source SET name = 'Soracom' WHERE type = 'harvest-backend-datasource' AND name = 'Harvest';`
	mg.AddMigration("rename soracom datasource name", NewRawSQLMigration(sql))
}
