package storage

import (
	"log"
)

func initDB() {
	sqlTable := `
		CREATE TABLE IF NOT EXISTS "pastebin" (
			"id" SERIAL PRIMARY KEY,
			"paste_id" varchar(12) NOT NULL UNIQUE,
			"user_id" varchar(256),
			"title" varchar(50),
			"content" bytea NOT NULL,
      "tags" varchar(15) ARRAY[15],
      "format" varchar(5) NOT NULL,
      "date" timestamp with time zone NOT NULL,
      "zlib" bool NOT NULL,
      "gdriveid" varchar(384),
      "gdrivedl" varchar(384)
		);
	`

	if _, err := DB.Exec(sqlTable); err != nil {
		log.Println("ERROR: The 'pastebin' table could not be initialised.")
		log.Fatalf("ERROR: %v", err)
	}

	sqlIndex := `
		CREATE INDEX IF NOT EXISTS paste_id_index ON pastebin(paste_id);
	`

	if _, err := DB.Exec(sqlIndex); err != nil {
		log.Println("ERROR: The 'pastebin.paste_id' index could not be initialised.")
		log.Fatalf("ERROR: %v", err)
	}
}

func init() {
	OnReady(initDB)
}
