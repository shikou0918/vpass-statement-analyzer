package database

import (
	"strings"
	"testing"
)

func TestMigrateCreatesUniqueCategoryRuleIndex(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := Open("file:" + dbName + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	first := CategoryRuleModel{MatchType: "contains", Pattern: "ローソン", CategoryID: 1, Priority: 1}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first category rule: %v", err)
	}
	duplicate := CategoryRuleModel{MatchType: "contains", Pattern: "ローソン", CategoryID: 1, Priority: 2}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected duplicate category rule insert to fail")
	}
	differentCategory := CategoryRuleModel{MatchType: "contains", Pattern: "ローソン", CategoryID: 2, Priority: 3}
	if err := db.Create(&differentCategory).Error; err != nil {
		t.Fatalf("same pattern can be used for a different category: %v", err)
	}
}
