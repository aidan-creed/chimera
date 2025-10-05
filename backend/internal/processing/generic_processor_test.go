package processing

import (
	"context"
	"testing"

	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/stretchr/testify/assert"
)

// Mock Querier for testing 'exists_in_items'
type mockQuerier struct {
	repository.Querier
	itemExists bool
}

func (m *mockQuerier) ItemExistsByBusinessKey(ctx context.Context, arg repository.ItemExistsByBusinessKeyParams) (int32, error) {
	if m.itemExists {
		return 1, nil
	}
	return 0, nil
}

func TestProcessRowValidation(t *testing.T) {
	// --- Test Setup ---
	testConfig := IngestionConfig{
		ReportType:  "TEST_VALIDATION",
		ItemType:    "TEST_ITEM",
		ScopeField:  "department",
		BusinessKey: []string{"employee_id"},
		ColumnMappings: []ColumnMapping{
			{
				CSVHeader: "employee_id",
				JSONField: "employee_id",
				Validation: ValidationRule{
					Required: true,
				},
			},
			{
				CSVHeader: "status",
				JSONField: "status",
				Validation: ValidationRule{
					Required: true,
					Enum:     []string{"ACTIVE", "INACTIVE", "TERMINATED"},
				},
			},
			{
				CSVHeader: "email",
				JSONField: "email",
				Validation: ValidationRule{
					Required: true,
					Regex:    `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
				},
			},
			{
				CSVHeader: "manager_id",
				JSONField: "manager_id",
				Validation: ValidationRule{
					ExistsInItems: "USER_PROFILE",
				},
			},
		},
	}

	processor := NewGenericProcessor(testConfig)
	ctx := context.Background()

	// --- Test Cases ---
	testCases := []struct {
		name          string
		record        []string
		headerMap     map[string]int
		mockQuerier   *mockQuerier
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid Record - Success",
			record:      []string{"123", "ACTIVE", "test@example.com", "manager1"},
			headerMap:   map[string]int{"employee_id": 0, "status": 1, "email": 2, "manager_id": 3},
			mockQuerier: &mockQuerier{itemExists: true},
			expectError: false,
		},
		{
			name:          "Invalid - Missing Required Field",
			record:        []string{"", "ACTIVE", "test@example.com", "manager1"},
			headerMap:     map[string]int{"employee_id": 0, "status": 1, "email": 2, "manager_id": 3},
			mockQuerier:   &mockQuerier{itemExists: true},
			expectError:   true,
			errorContains: "validation failed for column 'employee_id'",
		},
		{
			name:          "Invalid - Enum Validation Failure",
			record:        []string{"123", "PENDING", "test@example.com", "manager1"},
			headerMap:     map[string]int{"employee_id": 0, "status": 1, "email": 2, "manager_id": 3},
			mockQuerier:   &mockQuerier{itemExists: true},
			expectError:   true,
			errorContains: "is not in the allowed list",
		},
		{
			name:          "Invalid - Regex Validation Failure",
			record:        []string{"123", "ACTIVE", "not-an-email", "manager1"},
			headerMap:     map[string]int{"employee_id": 0, "status": 1, "email": 2, "manager_id": 3},
			mockQuerier:   &mockQuerier{itemExists: true},
			expectError:   true,
			errorContains: "does not match regex pattern",
		},
		{
			name:          "Invalid - ExistsInItems Failure",
			record:        []string{"123", "ACTIVE", "test@example.com", "nonexistent_manager"},
			headerMap:     map[string]int{"employee_id": 0, "status": 1, "email": 2, "manager_id": 3},
			mockQuerier:   &mockQuerier{itemExists: false}, // Mock that the manager does NOT exist
			expectError:   true,
			errorContains: "does not exist as a business_key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := processor.processRow(ctx, tc.record, tc.headerMap, tc.mockQuerier)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
