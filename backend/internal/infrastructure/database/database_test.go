package database

import (
	"strings"
	"testing"
)

func TestMigrateDeduplicatesCategoryRulesBeforeUniqueIndex(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := Open("file:" + dbName + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE category_rule_models (
			id integer PRIMARY KEY AUTOINCREMENT,
			match_type text,
			pattern text,
			category_id integer,
			priority integer
		)
	`).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO category_rule_models (match_type, pattern, category_id, priority)
		VALUES
			('contains', 'ローソン', 1, 1),
			('contains', 'ローソン', 1, 2),
			('contains', 'ローソン', 2, 3)
	`).Error; err != nil {
		t.Fatalf("insert legacy duplicates: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	var count int64
	if err := db.Model(&CategoryRuleModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count category rules: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected exact duplicate to be removed, got %d rules", count)
	}

	err = db.Create(&CategoryRuleModel{MatchType: "contains", Pattern: "ローソン", CategoryID: 1, Priority: 4}).Error
	if err == nil {
		t.Fatal("expected duplicate category rule insert to fail")
	}
}
