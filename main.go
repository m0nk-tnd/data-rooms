package main

import (
	"github.com/go-pg/pg/v10"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

var db *pg.DB

func handleRequests(db *pg.DB) {
	r := mux.NewRouter()
	r.HandleFunc("/intapi/datarooms", dataRooms)
	r.HandleFunc("/intapi/datarooms/{roomId}", dataRoomDelete)
	r.HandleFunc("/intapi/content", dataRoomFilesList)
	r.HandleFunc("/intapi/upload", dataRoomFilesUpload)
	r.HandleFunc("/intapi/download", dataRoomFilesDownloadByMeta)
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe("localhost:5001", nil))

}

func createDirectories() {
	if _, err := os.Stat("tmp"); os.IsNotExist(err) {
		os.Mkdir("tmp", os.ModePerm)
	}

	if _, err := os.Stat("files"); os.IsNotExist(err) {
		os.Mkdir("files", os.ModePerm)
	}
}

func main() {
	db = NewDBConn()
	defer db.Close()

	createDirectories()

	err := createSchemas(db)
	if err != nil {
		panic(err)
	}

	handleRequests(db)
}
