package pkg

import (
	"context"
	"database/sql"

	"github.com/destrex271/postgresexporter/internal"
)

const (
	AttributesMappingTableName = internal.AttributesMappingTableName
	AttributesMappingAttributeFieldName = internal.AttributesMappingAttributeFieldName
)

type AttributesMapping = internal.AttributesMapping

func GetAttributesMappingByName(ctx context.Context, client *sql.DB, schemaName string, name string) (AttributesMapping, error) {
	return internal.GetAttributesMappingByName(ctx, client, schemaName, name)
}

func GetAttributesMappingsByNames(ctx context.Context, client *sql.DB, schemaName string, names []string) ([]AttributesMapping, error) {
	return internal.GetAttributesMappingsByNames(ctx, client, schemaName, names)
}

func GetAttributesValueAndFieldNameMap(attrsMapping *AttributesMapping) (map[string]string, error) {
	return internal.GetAttributesValueAndFieldNameMap(attrsMapping)
}
