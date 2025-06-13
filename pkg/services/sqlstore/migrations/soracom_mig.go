package migrations

import . "github.com/grafana/grafana/pkg/services/sqlstore/migrator"

func addSoracomDatasourceNameMigration(mg *Migrator) {
	mg.AddMigration("rename soracom datasource name", NewRawSQLMigration(`UPDATE data_source SET name = 'Soracom' WHERE type = 'harvest-backend-datasource' AND name = 'Harvest';`))
	mg.AddMigration("rename soracom datasource secrets namespace", NewRawSQLMigration(`UPDATE secrets SET namespace = 'Soracom' WHERE type = 'datasource' AND namespace = 'Harvest';`))
}
