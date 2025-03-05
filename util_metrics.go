package postgresexporter

type DBType string

const (
	DBTypePostgreSQL  DBType = "postgresql"
	DBTypeTimescaleDB DBType = "timescaledb"
	DBTypeParadeDB    DBType = "paradedb"
)
