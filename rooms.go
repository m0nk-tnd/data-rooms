package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

// models

type DataRoom struct {
	tableName    struct{}        `pg:"data_rooms"`
	Id           uuid.UUID       `json:"roomID" pg:"id,pk,type:uuid,default:uuid_generate_v4()"`
	Name         string          `json:"roomName" pg:"name"`
	Size         int64           `json:"roomSize" pg:"size"`
	Expires      time.Time       `json:"roomExpires" pg:"expired"`
	RootFolderID uuid.UUID       `json:"rootFolderID" pg:"root_folder_id,type:uuid,unique,default:uuid_generate_v4()"`
	Files        []*DataRoomFile `json:"-" pg:"rel:has-many,fk:root_folder_id,join_fk:folder_id"`
}

func (d *DataRoom) roomSizeUsed() (sum int) {
	for _, file := range d.Files {
		sum += file.FileSize
	}
	return
}

func (d *DataRoom) roomNumFiles() int {
	return len(d.Files)
}

func (d *DataRoom) MarshalJSON() ([]byte, error) {
	type Alias DataRoom
	return json.Marshal(&struct {
		*Alias
		Expires      string `json:"roomExpires"`
		RoomSizeUsed int    `json:"roomSizeUsed"`
		RoomNumFiles int    `json:"roomNumFiles"`
	}{
		Alias:        (*Alias)(d),
		Expires:      d.Expires.Format(time.RFC3339),
		RoomSizeUsed: d.roomSizeUsed(),
		RoomNumFiles: d.roomNumFiles(),
	})
}

// views

func dataRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var rooms []DataRoom
		err := db.Model(&rooms).Select()
		if err != nil {
			panic(err)
		}

		res, err := json.Marshal(rooms)
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		fmt.Println("Welcome to the dataRooms page!")

	case http.MethodPost:
		var obj DataRoom

		err := json.NewDecoder(r.Body).Decode(&obj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// default fields
		obj.Id = uuid.New()
		obj.Expires = time.Now().AddDate(0, 1, 0)
		obj.RootFolderID = uuid.New()

		_, err = db.Model(obj).Insert()
		if err != nil {
			panic(err)
		}

		res, err := json.Marshal(obj)
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		fmt.Println(fmt.Sprintf("created new room: <%s %s>", obj.Id, obj.Name))

	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}

func dataRoomDelete(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodDelete:
		vars := mux.Vars(r)
		roomId := vars["roomId"]

		// TODO add condition if there are files in room
		room := &DataRoom{Id: uuid.MustParse(roomId)}
		deleted, err := db.Model(room).WherePK().Delete()
		if err != nil {
			panic(err)
		}

		res := fmt.Sprintf("{\"deleted\": %d}", deleted.RowsAffected())
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(res))
		fmt.Println(fmt.Sprintf("deleted: <%s>", roomId))

	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}

}
