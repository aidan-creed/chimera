package processing

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// TransformFunc defines the signature for any transformation function.
// It now accepts an optional argument string.
type TransformFunc func(input interface{}, arg string) (interface{}, error)

// ValidationFunc defines the signature for any validation function.
type ValidationFunc func(ctx context.Context, queries repository.Querier, input interface{}, rule ValidationRule) error

var transformRegistry = make(map[string]TransformFunc)
var validationRegistry = make(map[string]ValidationFunc)

// init runs when the package is loaded, registering our built-in functions
func init() {
	// Register Transformations
	transformRegistry["trim_space"] = transformTrimSpace
	transformRegistry["to_uppercase"] = transformToUppercase
	transformRegistry["to_integer"] = transformToInteger
	transformRegistry["to_decimal"] = transformToDecimal
	transformRegistry["to_date"] = transformToDate

	// Register Validations
	validationRegistry["required"] = validationRequired
	validationRegistry["enum"] = validateEnum
	validationRegistry["regex"] = validateRegex
	validationRegistry["exists_in_items"] = validateExistsInItems
}

// --- Transformation Implementations ---

func transformTrimSpace(input interface{}, arg string) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("trim_space requires a string input")
	}
	return strings.TrimSpace(str), nil
}

func transformToUppercase(input interface{}, arg string) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("to_uppercase requires a string input")
	}
	return strings.ToUpper(str), nil
}

func transformToInteger(input interface{}, arg string) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("to_integer requires a string input")
	}

	cleanStr := strings.ReplaceAll(str, ",", "")
	cleanStr = strings.TrimSpace(cleanStr)

	if cleanStr == "" {
		return nil, nil
	}

	i, err := strconv.ParseInt(cleanStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as integer: %w", str, err)
	}
	return i, nil
}

func transformToDecimal(input interface{}, arg string) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("to_decimal requires a string input")
	}
	d, err := decimal.NewFromString(str)
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as decimal: %w", str, err)
	}
	return d, nil
}

func transformToDate(input interface{}, arg string) (interface{}, error) {
	layout := arg
	if layout == "" {
		// Default to common format if no layout is provided in YAML
		layout = "2006-01-02"
	}
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("to_date requires a string input")
	}
	t, err := time.ParseInLocation(layout, str, time.UTC)
	if err != nil {
		return nil, fmt.Errorf("could not parse date '%s' with format '%s' in UTC: %w", str, layout, err)
	}
	return t, nil
}

// --- Validation Implementaton ---

func validationRequired(ctx context.Context, queries repository.Querier, input interface{}, rule ValidationRule) error {
	if !rule.Required {
		return nil
	}
	if input == nil {
		return fmt.Errorf("is a required field")
	}

	allowZero := true
	if rule.AllowZero != nil && !*rule.AllowZero {
		allowZero = false
	}

	switch v := input.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("is a required field")
		}
	case int, int32, int64:
		var val int64
		switch i := v.(type) {
		case int:
			val = int64(i)
		case int32:
			val = int64(i)
		case int64:
			val = i
		}
		if !allowZero && val == 0 {
			return fmt.Errorf("is a required field and zero is not an allowed value")
		}

	case decimal.Decimal:
		if !allowZero && v.IsZero() {
			return fmt.Errorf("is a required field and zero is not an allowed value")
		}
	case time.Time:
		if v.IsZero() {
			return fmt.Errorf("is a required field")
		}
	}
	return nil
}

func validateEnum(ctx context.Context, queries repository.Querier, input interface{}, rule ValidationRule) error {
	if len(rule.Enum) == 0 {
		return nil
	}
	str, ok := input.(string)
	if !ok {
		return fmt.Errorf("value must be a string to be checked against an enum")
	}
	for _, allowedValue := range rule.Enum {
		if str == allowedValue {
			return nil
		}
	}
	return fmt.Errorf("value '%s' is not in the allowed list: %v", str, rule.Enum)
}

func validateRegex(ctx context.Context, queries repository.Querier, input interface{}, rule ValidationRule) error {
	if rule.Regex == "" {
		return nil
	}
	re, err := regexp.Compile(rule.Regex)
	if err != nil {
		return fmt.Errorf("invalid regex pattern in config: %s", rule.Regex)
	}
	str, ok := input.(string)
	if !ok {
		return fmt.Errorf("value must be a string to be matched against a regex")
	}
	if !re.MatchString(str) {
		return fmt.Errorf("value '%s' does not match regex pattern '%s'", str, rule.Regex)
	}
	return nil
}

func validateExistsInItems(ctx context.Context, queries repository.Querier, input interface{}, rule ValidationRule) error {
	if rule.ExistsInItems == "" {
		return nil
	}
	itemTypeToCheck := repository.ItemType(rule.ExistsInItems)
	if itemTypeToCheck == "" {
		return fmt.Errorf("exists_in_items validator requires an argument specifying the item_type to check against")
	}

	value, ok := input.(string)
	if !ok {
		return fmt.Errorf("exists_in_items can only validate string fields")
	}
	if value == "" {
		return nil
	}

	params := repository.ItemExistsByBusinessKeyParams{
		ItemType:    itemTypeToCheck,
		BusinessKey: pgtype.Text{String: value, Valid: true},
	}

	exists, err := queries.ItemExistsByBusinessKey(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Database error during exists_in_items validation", "error", err)
		return fmt.Errorf("database error checking existence of %s", value)
	}

	if exists == 0 {
		return fmt.Errorf("value '%s' does not exist as a business_key for item_type '%s'", value, itemTypeToCheck)
	}

	return nil
}
