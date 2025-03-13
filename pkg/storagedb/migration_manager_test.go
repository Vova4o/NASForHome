package storagedb

import (
	"database/sql"
	"errors"
	"fmt" // Добавьте этот импорт
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Сохраняем оригинальные миграции
var originalMigrations []Migration

// TestMain используется для настройки и очистки тестов
func TestMain(m *testing.M) {
	// Сохраняем оригинальные миграции
	originalMigrations = migrations

	// Запускаем тесты
	m.Run()

	// Восстанавливаем оригинальные миграции после всех тестов
	migrations = originalMigrations
}

// TestResolveTargetVersion проверяет функцию resolveTargetVersion
func TestResolveTargetVersion(t *testing.T) {
	// Создаем тестовые миграции
	testMigrations := []Migration{
		{Version: 1, Description: "Test Migration 1"},
		{Version: 2, Description: "Test Migration 2"},
		{Version: 3, Description: "Test Migration 3"},
	}

	// Устанавливаем тестовые миграции
	migrations = testMigrations
	defer func() { migrations = originalMigrations }()

	// Тестовые кейсы
	tests := []struct {
		name           string
		targetVersion  int
		expectedResult int
	}{
		{"Specific version", 2, 2},
		{"All migrations (-1)", -1, 3},
		{"Zero version", 0, 0},
		{"Version higher than available", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTargetVersion(tt.targetVersion)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// TestApplyMigrations проверяет функцию applyMigrations
func TestApplyMigrations(t *testing.T) {
	// Создаем тестовую БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые миграции с реальными функциями
	var migrationsApplied []int
	testMigrations := []Migration{
		{
			Version:     1,
			Description: "Test Migration 1",
			Up: func(db *sql.DB) error {
				migrationsApplied = append(migrationsApplied, 1)
				return nil
			},
		},
		{
			Version:     2,
			Description: "Test Migration 2",
			Up: func(db *sql.DB) error {
				migrationsApplied = append(migrationsApplied, 2)
				return nil
			},
		},
		{
			Version:     3,
			Description: "Test Migration 3",
			Up: func(db *sql.DB) error {
				migrationsApplied = append(migrationsApplied, 3)
				return nil
			},
		},
	}

	// Устанавливаем тестовые миграции
	migrations = testMigrations
	defer func() { migrations = originalMigrations }()

	// Настраиваем ожидания для записи версий в БД
	mock.ExpectExec("INSERT INTO migrations").WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO migrations").WithArgs(2).WillReturnResult(sqlmock.NewResult(2, 1))

	// Вызываем тестируемую функцию (мигрируем с 0 до 2)
	err = applyMigrations(db, 0, 2)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, migrationsApplied)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestApplyMigrationsError проверяет обработку ошибок при миграции
func TestApplyMigrationsError(t *testing.T) {
	// Создаем тестовую БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые миграции с ошибкой в одной из миграций
	testMigrations := []Migration{
		{
			Version:     1,
			Description: "Test Migration 1",
			Up: func(db *sql.DB) error {
				return nil
			},
		},
		{
			Version:     2,
			Description: "Test Migration with Error",
			Up: func(db *sql.DB) error {
				return errors.New("тестовая ошибка миграции")
			},
		},
	}

	// Устанавливаем тестовые миграции
	migrations = testMigrations
	defer func() { migrations = originalMigrations }()

	// Настраиваем ожидания для записи версий в БД
	mock.ExpectExec("INSERT INTO migrations").WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))

	// Вызываем тестируемую функцию (мигрируем с 0 до 2)
	err = applyMigrations(db, 0, 2)

	// Проверяем результаты
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "тестовая ошибка миграции")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRollbackMigrations проверяет функцию rollbackMigrations
func TestRollbackMigrations(t *testing.T) {
	// Создаем тестовую БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые миграции с реальными функциями отката
	var migrationsRolledBack []int
	testMigrations := []Migration{
		{
			Version:     1,
			Description: "Test Migration 1",
			Down: func(db *sql.DB) error {
				migrationsRolledBack = append(migrationsRolledBack, 1)
				return nil
			},
		},
		{
			Version:     2,
			Description: "Test Migration 2",
			Down: func(db *sql.DB) error {
				migrationsRolledBack = append(migrationsRolledBack, 2)
				return nil
			},
		},
		{
			Version:     3,
			Description: "Test Migration 3",
			Down: func(db *sql.DB) error {
				migrationsRolledBack = append(migrationsRolledBack, 3)
				return nil
			},
		},
	}

	// Устанавливаем тестовые миграции
	migrations = testMigrations
	defer func() { migrations = originalMigrations }()

	// Настраиваем ожидания для удаления версий из БД
	mock.ExpectExec("DELETE FROM migrations WHERE version = \\$1").WithArgs(3).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM migrations WHERE version = \\$1").WithArgs(2).WillReturnResult(sqlmock.NewResult(0, 1))

	// Вызываем тестируемую функцию (откат с 3 до 1)
	err = rollbackMigrations(db, 3, 1)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, []int{3, 2}, migrationsRolledBack) // Откат идет в обратном порядке
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRollbackMigrationsError проверяет обработку ошибок при откате
func TestRollbackMigrationsError(t *testing.T) {
	// Создаем тестовую БД
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые миграции с ошибкой в одной из миграций
	testMigrations := []Migration{
		{
			Version:     1,
			Description: "Test Migration 1",
			Down: func(db *sql.DB) error {
				return nil
			},
		},
		{
			Version:     2,
			Description: "Test Migration 2",
			Down: func(db *sql.DB) error {
				return errors.New("тестовая ошибка отката")
			},
		},
	}

	// Устанавливаем тестовые миграции
	migrations = testMigrations
	defer func() { migrations = originalMigrations }()

	// Вызываем тестируемую функцию (откат с 2 до 0)
	err = rollbackMigrations(db, 2, 0)

	// Проверяем результаты
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "тестовая ошибка отката")
}

// TestMigrateDatabase проверяет основную функцию миграции
func TestMigrateDatabase(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion int
		targetVersion  int
		expectApply    bool
		expectRollback bool
	}{
		{"Apply migrations", 1, 3, true, false},
		{"Rollback migrations", 3, 1, false, true},
		{"Same version", 2, 2, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовую БД
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			// Настраиваем ожидания для получения текущей версии
			mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT COALESCE").
				WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(tt.currentVersion))

			// Создаем замыкания с корректным захватом переменных
			migratedUp := false   // Выносим в локальную область теста
			migratedDown := false // Выносим в локальную область теста

			// Явные счетчики вызовов для мониторинга
			upCallCount := 0
			downCallCount := 0

			// Создаем простые тестовые миграции
			testMigrations := []Migration{
				{
					Version:     1,
					Description: "Test Migration 1",
					Up: func(db *sql.DB) error {
						migratedUp = true
						upCallCount++
						return nil
					},
					Down: func(db *sql.DB) error {
						fmt.Println("Вызван откат миграции 1")
						migratedDown = true
						downCallCount++
						return nil
					},
				},
				{
					Version:     2,
					Description: "Test Migration 2",
					Up: func(db *sql.DB) error {
						fmt.Println("Применение миграции 2") // Добавьте эту строку для отладки
						migratedUp = true                    // Добавьте эту строку
						upCallCount++                        // Добавьте эту строку
						return nil
					},
					Down: func(db *sql.DB) error {
						fmt.Println("Вызван откат миграции 2")
						migratedDown = true
						downCallCount++
						return nil
					},
				},
				{
					Version:     3,
					Description: "Test Migration 3",
					Up: func(db *sql.DB) error {
						fmt.Println("Применение миграции 3") // Добавьте эту строку для отладки
						migratedUp = true                    // Добавьте эту строку
						upCallCount++                        // Добавьте эту строку
						return nil
					},
					Down: func(db *sql.DB) error {
						fmt.Println("Вызван откат миграции 3")
						migratedDown = true
						downCallCount++
						return nil
					},
				},
			}

			// Устанавливаем тестовые миграции
			migrations = testMigrations

			// Дополнительные ожидания в зависимости от сценария
			if tt.expectApply {
				for v := tt.currentVersion + 1; v <= tt.targetVersion; v++ {
					mock.ExpectExec("INSERT INTO migrations").
						WithArgs(v).
						WillReturnResult(sqlmock.NewResult(int64(v), 1))
				}
			} else if tt.expectRollback {
				for v := tt.currentVersion; v > tt.targetVersion; v-- {
					mock.ExpectExec("DELETE FROM migrations").
						WithArgs(v).
						WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}

			// Вызываем тестируемую функцию
			err = MigrateDatabase(db, tt.targetVersion)

			// Добавьте для отладки:
			if tt.expectRollback && !migratedDown {
				t.Logf("Ожидался откат, но migratedDown = false. currentVersion: %d, targetVersion: %d",
					tt.currentVersion, tt.targetVersion)
			}

			// Проверяем результаты
			assert.NoError(t, err)
			if tt.expectApply {
				assert.True(t, migratedUp, "Должны быть применены миграции")
				assert.GreaterOrEqual(t, upCallCount, 1, "Должны быть применены миграции")
			}
			if tt.expectRollback {
				assert.True(t, migratedDown, "Должен быть выполнен откат миграций")
				assert.GreaterOrEqual(t, downCallCount, 1, "Должен быть выполнен откат миграций")
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
