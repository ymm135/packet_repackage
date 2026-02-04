package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Open database
	db, err := gorm.Open(sqlite.Open("../data/packet.db"), &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Check if migration is needed
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('rules') WHERE name='output_template'").Scan(&count)

	if count > 0 {
		fmt.Println("Migrating database: renaming output_template to output_options...")

		// Rename column
		err = db.Exec("ALTER TABLE rules RENAME COLUMN output_template TO output_options").Error
		if err != nil {
			log.Fatal("Failed to migrate:", err)
		}

		fmt.Println("Migration completed successfully!")
	} else {
		fmt.Println("Migration not needed - column already renamed or doesn't exist")
	}
}
