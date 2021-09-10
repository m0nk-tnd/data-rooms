package main

import (
	"github.com/google/uuid"
	"net/http"
	"time"
)

// models

type Ticket struct {
	tableName   struct{}      `pg:"tickets"`
	Id          uuid.UUID     `json:"id" pg:"id,pk,type:uuid,default:uuid_generate_v4()"`
	IsAvailable bool          `pg:"is_available"`
	Created     time.Time     `pg:"created"`
	FileId      uuid.UUID     `pg:"file_id,type:uuid"`
	File        *DataRoomFile `pg:"rel:has-many,fk:file_id"`
}

// views

func dataRoomFilesCreateTicket(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:

	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}
