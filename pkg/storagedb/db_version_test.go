package storagedb

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMigrationsTable(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "Ошибка создания мока базы данных")
	defer db.Close()

	// Настраиваем ожидание выполнения запроса создания таблицы
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Вызываем тестируемую функцию
	err = createMigrationsTable(db)

	// Проверяем результаты
	assert.NoError(t, err, "Создание таблицы миграций должно пройти успешно")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые запросы должны быть выполнены")
}

func TestCreateMigrationsTableError(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Настраиваем ожидание ошибки при создании таблицы
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnError(errors.New("ошибка создания таблицы"))

	// Вызываем тестируемую функцию
	err = createMigrationsTable(db)

	// Проверяем результаты
	assert.Error(t, err, "Ожидается ошибка при создании таблицы")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCurrentVersionEmpty(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Настраиваем ожидания создания таблицы и запроса версии
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Ожидаем запрос на получение версии, возвращаем 0 (пустая таблица)
	mock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(0))

	// Вызываем тестируемую функцию
	version, err := getCurrentVersion(db)

	// Проверяем результаты
	assert.NoError(t, err, "Получение версии должно пройти без ошибок")
	assert.Equal(t, 0, version, "Версия для пустой таблицы должна быть 0")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCurrentVersionWithValue(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Настраиваем ожидания
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Ожидаем запрос на получение версии, возвращаем 5
	mock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(5))

	// Вызываем тестируемую функцию
	version, err := getCurrentVersion(db)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, 5, version, "Версия должна соответствовать значению из базы")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCurrentVersionError(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Настраиваем ожидания создания таблицы
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Ожидаем ошибку при выполнении запроса версии
	mock.ExpectQuery("SELECT COALESCE").
		WillReturnError(errors.New("ошибка получения версии"))

	// Вызываем тестируемую функцию
	_, err = getCurrentVersion(db)

	// Проверяем результаты
	assert.Error(t, err, "Должна быть ошибка при получении версии")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateVersion(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Настраиваем ожидание запроса на добавление версии
	mock.ExpectExec("INSERT INTO migrations").
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Вызываем тестируемую функцию
	err = updateVersion(db, 10)

	// Проверяем результаты
	assert.NoError(t, err, "Обновление версии должно пройти успешно")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateVersionError(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Настраиваем ожидание ошибки при добавлении версии
	mock.ExpectExec("INSERT INTO migrations").
		WithArgs(10).
		WillReturnError(errors.New("ошибка добавления версии"))

	// Вызываем тестируемую функцию
	err = updateVersion(db, 10)

	// Проверяем результаты
	assert.Error(t, err, "Должна быть ошибка при обновлении версии")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEndToEndVersionFlow(t *testing.T) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 1. Создаем таблицу миграций
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 2. Получаем начальную версию (0)
	mock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(0))

	// 3. Обновляем версию до 1
	mock.ExpectExec("INSERT INTO migrations").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 4. Снова получаем версию (теперь должна быть 1)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(1))

	// Выполняем тестовый сценарий
	version, err := getCurrentVersion(db)
	assert.NoError(t, err)
	assert.Equal(t, 0, version)

	err = updateVersion(db, 1)
	assert.NoError(t, err)

	version, err = getCurrentVersion(db)
	assert.NoError(t, err)
	assert.Equal(t, 1, version)

	assert.NoError(t, mock.ExpectationsWereMet())
}
