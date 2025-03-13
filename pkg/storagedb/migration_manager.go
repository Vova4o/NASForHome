package storagedb

import (
	"database/sql"
	"fmt"
)

// MigrateDatabase выполняет миграцию до указанной версии (-1 = все)
func MigrateDatabase(db *sql.DB, targetVersion int) error {
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return err
	}

	targetVersion = resolveTargetVersion(targetVersion)

	if targetVersion > currentVersion {
		return applyMigrations(db, currentVersion, targetVersion)
	} else if targetVersion < currentVersion {
		return rollbackMigrations(db, currentVersion, targetVersion)
	}
	return nil
}

// Вычисление целевой версии
func resolveTargetVersion(targetVersion int) int {
	if targetVersion == -1 && len(migrations) > 0 {
		return migrations[len(migrations)-1].Version
	}
	return targetVersion
}

// Применение миграций
func applyMigrations(db *sql.DB, currentVersion, targetVersion int) error {
	for _, migration := range migrations {
		if migration.Version > currentVersion && migration.Version <= targetVersion {
			fmt.Printf("Применение миграции %d: %s\n", migration.Version, migration.Description)
			if err := migration.Up(db); err != nil {
				return fmt.Errorf("ошибка миграции %d: %w", migration.Version, err)
			}

			// Обновляем версию в БД
			if err := updateVersion(db, migration.Version); err != nil {
				return fmt.Errorf("ошибка обновления версии %d: %w", migration.Version, err)
			}

			fmt.Printf("✔️ Миграция %d успешно применена\n", migration.Version)
		}
	}
	return nil
}

// Откат миграций
func rollbackMigrations(db *sql.DB, currentVersion, targetVersion int) error {
    for i := len(migrations) - 1; i >= 0; i-- {
        migration := migrations[i]
        if migration.Version <= currentVersion && migration.Version > targetVersion {
            fmt.Printf("Откат миграции %d: %s\n", migration.Version, migration.Description)
            if err := migration.Down(db); err != nil {
                return fmt.Errorf("ошибка отката %d: %w", migration.Version, err)
            }
            if _, err := db.Exec("DELETE FROM migrations WHERE version = $1", migration.Version); err != nil {
                return fmt.Errorf("ошибка удаления версии %d: %w", migration.Version, err)
            }
            fmt.Println("✔️ Миграция успешно отменена")
        }
    }
    return nil 
}
