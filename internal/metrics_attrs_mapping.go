package internal

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	AttributesMappingTableName = "_attributes_mappings"
	AttributesMappingAttributeFieldName = "Attribute"

	attributesMappingInsertSQL = `
	INSERT INTO "%s"."%s" (name) VALUES ($1) ON CONFLICT (name) DO NOTHING
	`

	attributesMappingUpdateSQL = `
	UPDATE "%s"."%s"
		SET attribute1 = $2,   attribute2 = $3,   attribute3 = $4,   attribute4 = $5,   attribute5 = $6,
			attribute6 = $7,   attribute7 = $8,   attribute8 = $9,   attribute9 = $10,  attribute10 = $11,
			attribute11 = $12, attribute12 = $13, attribute13 = $14, attribute14 = $15, attribute15 = $16,
			attribute16 = $17, attribute17 = $18, attribute18 = $19, attribute19 = $20, attribute20 = $21
		WHERE name = $1
	`
)

var (
	attributesMappingTableColumns = []string{
		"name VARCHAR PRIMARY KEY",

		"attribute1  VARCHAR",
		"attribute2  VARCHAR",
		"attribute3  VARCHAR",
		"attribute4  VARCHAR",
		"attribute5  VARCHAR",
		"attribute6  VARCHAR",
		"attribute7  VARCHAR",
		"attribute8  VARCHAR",
		"attribute9  VARCHAR",
		"attribute10 VARCHAR",
		"attribute11 VARCHAR",
		"attribute12 VARCHAR",
		"attribute13 VARCHAR",
		"attribute14 VARCHAR",
		"attribute15 VARCHAR",
		"attribute16 VARCHAR",
		"attribute17 VARCHAR",
		"attribute18 VARCHAR",
		"attribute19 VARCHAR",
		"attribute20 VARCHAR",
	}
)

type AttributesMapping struct {
	Name string `db:"name"`

	Attribute1  string `db:"attribute1"`
	Attribute2  string `db:"attribute2"`
	Attribute3  string `db:"attribute3"`
	Attribute4  string `db:"attribute4"`
	Attribute5  string `db:"attribute5"`
	Attribute6  string `db:"attribute6"`
	Attribute7  string `db:"attribute7"`
	Attribute8  string `db:"attribute8"`
	Attribute9  string `db:"attribute9"`
	Attribute10 string `db:"attribute10"`
	Attribute11 string `db:"attribute11"`
	Attribute12 string `db:"attribute12"`
	Attribute13 string `db:"attribute13"`
	Attribute14 string `db:"attribute14"`
	Attribute15 string `db:"attribute15"`
	Attribute16 string `db:"attribute16"`
	Attribute17 string `db:"attribute17"`
	Attribute18 string `db:"attribute18"`
	Attribute19 string `db:"attribute19"`
	Attribute20 string `db:"attribute20"`
}

func CreateAttributesMappingTable(ctx context.Context, client *sql.DB, schemaName string) error {
	query := fmt.Sprintf(createTableIfNotExistsSQL,
		schemaName, AttributesMappingTableName, strings.Join(attributesMappingTableColumns, ","),
	)
	_, err := client.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed creating attribute mapping table: %w", err)
	}

	return nil
}

func insertAttributesMapping(ctx context.Context, client *sql.DB, schemaName string, attributesMapping *AttributesMapping) error {
	query := fmt.Sprintf(attributesMappingInsertSQL, schemaName, AttributesMappingTableName)
	_, err := client.ExecContext(ctx, query, attributesMapping.Name)

	return err
}

func updateAttributesMapping(ctx context.Context, client *sql.DB, schemaName string, attributesMapping *AttributesMapping) error {
	query := fmt.Sprintf(attributesMappingUpdateSQL, schemaName, AttributesMappingTableName)
	args := extractArgs(attributesMapping)
	_, err := client.ExecContext(ctx, query, args...)

	return err
}

func extractArgs(attrsMapping *AttributesMapping) []any {
	attrsMappingVal := reflect.Indirect(reflect.ValueOf(attrsMapping))

	args := make([]any, attrsMappingVal.NumField())

	for i := range attrsMappingVal.NumField() {
		field := attrsMappingVal.Field(i)

		if field.IsValid() && !field.IsZero() {
			args[i] = field.String()
		} else {
			args[i] = nil
		}
	}

	return args
}

func GetAttributesMappingByName(ctx context.Context, client *sql.DB, schemaName string, name string) (AttributesMapping, error) {
	attributesMappings, err := GetAttributesMappingsByNames(ctx, client, schemaName, []string{name})
	if err != nil {
		return AttributesMapping{}, err
	}

	if len(attributesMappings) == 0 {
		return AttributesMapping{}, fmt.Errorf("attributes mapping not found")
	} else if len(attributesMappings) == 1 {
		return attributesMappings[0], nil
	}

	return AttributesMapping{}, fmt.Errorf("multiple attributes mappings were found")
}

func GetAttributesMappingsByNames(ctx context.Context, client *sql.DB, schemaName string, names []string) ([]AttributesMapping, error) {
	query := `SELECT * FROM "%s"."%s" WHERE name = ANY($1)`
	rows, err := client.QueryContext(ctx, fmt.Sprintf(query, schemaName, AttributesMappingTableName), names)
	if err != nil {
		return nil, err
	}

	result := []AttributesMapping{}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		am := AttributesMapping{}

		values := make([]any, len(columns))
		for i := range values {
			values[i] = new(sql.Null[any])
		}

		err := rows.Scan(values...)
		if err != nil {
			return nil, err
		}

		setValuesToAttrsMappingFields(&am, values)

		result = append(result, am)
	}

	return result, nil
}

func setValuesToAttrsMappingFields(attrsMapping *AttributesMapping, values []any) error {
	attrsMappingVal := reflect.Indirect(reflect.ValueOf(attrsMapping))

	for i := range attrsMappingVal.NumField() {
		field := attrsMappingVal.Field(i)

		if !field.CanSet() {
			return fmt.Errorf("attribute field %s can't be set", field.String())
		}

		value, ok := values[i].(*sql.Null[any])
		if !ok {
			return fmt.Errorf("value can't be converted to sql.Null[any]")
		}

		if value != nil && value.Valid {
			field.Set(reflect.ValueOf(value.V).Convert(field.Type()))
		}
	}

	return nil
}

func GetAttributesValueAndFieldNameMap(attrsMapping *AttributesMapping) (map[string]string, error) {
	result := map[string]string{}

	attrsMappingVal := reflect.Indirect(reflect.ValueOf(attrsMapping))
	attrsMappingTyp := attrsMappingVal.Type()
	for n := range attrsMappingTyp.NumField() {
		fieldTyp := attrsMappingTyp.Field(n)
		fieldVal := attrsMappingVal.Field(n)

		if fieldVal.IsValid() {
			if !strings.HasPrefix(fieldTyp.Name, AttributesMappingAttributeFieldName) {
				continue
			}

			if !fieldVal.IsZero() {
				result[fieldVal.String()] = fieldTyp.Tag.Get("db")
			} else {
				break
			}
		} else {
			return nil, fmt.Errorf("invalid attributes mapping field value")
		}
	}

	return result, nil
}

func groupAttrsMappingsByName(attrsMappings []AttributesMapping) map[string]AttributesMapping {
	result := map[string]AttributesMapping{}

	for _, am := range attrsMappings {
		result[am.Name] = am
	}

	return result
}

func getAttrsNameAndPosMap(attrsMapping *AttributesMapping) (map[string]int, error) {
	result := map[string]int{}

	attrsMappingVal := reflect.Indirect(reflect.ValueOf(attrsMapping))
	for pos := 1; pos <= maxAttributesNumber; pos++ {
		field := attrsMappingVal.FieldByName(AttributesMappingAttributeFieldName + strconv.Itoa(pos))

		if field.IsValid() {
			if !field.IsZero() {
				result[field.String()] = pos
			} else {
				break
			}
		} else {
			return nil, fmt.Errorf("invalid attributes mapping field value")
		}
	}

	return result, nil
}

func findNextAvailableAttrPos(attrsMapping *AttributesMapping) (int, error) {
	attrsMappingVal := reflect.ValueOf(attrsMapping).Elem()
	for pos := 1; pos <= maxAttributesNumber; pos++ {
		field := attrsMappingVal.FieldByName(AttributesMappingAttributeFieldName + strconv.Itoa(pos))

		if field.IsZero() {
			return pos, nil
		}
	}

	return 0, fmt.Errorf("there are no more available attribute positions")
}

func setAttrValueByPos(attrsMapping *AttributesMapping, pos int, name string) error {
	attrsMappingVal := reflect.Indirect(reflect.ValueOf(attrsMapping))

	field := attrsMappingVal.FieldByName(AttributesMappingAttributeFieldName + strconv.Itoa(pos))
	if !field.CanSet() {
		return fmt.Errorf("attribute field can't be set")
	}

	field.Set(reflect.ValueOf(name))

	return nil
}
