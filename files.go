package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// models

type DataRoomFile struct {
	tableName   struct{}  `pg:"data_room_files"`
	FileMetaId  uuid.UUID `json:"fileMetaID" pg:"id,pk,type:uuid"`
	FileId      string    `json:"fileID" pg:"file_id"`
	FileHash    string    `json:"fileHash" pg:"file_hash"`
	FileName    string    `json:"fileName" pg:"name"`
	FileDesc    string    `json:"fileDesc" pg:"description"`
	FileCreated time.Time `json:"fileCreated" pg:"created"`
	FileExpires time.Time `json:"fileExpires" pg:"expired"`
	FileStatus  string    `json:"fileStatus" pg:"status"`
	FileSize    int       `json:"fileSize" pg:"size"`
	Chunks      int       `json:"-" pg:"chunks"`
	FolderId    uuid.UUID `json:"folderID" pg:"folder_id,type:uuid"`
	AuthorId    string    `json:"authorID" pg:"author_id,type:uuid"`
	Room        DataRoom  `json:"-" pg:"rel:has-one,fk:folder_id,join_fk:root_folder_id"`
	//AuthorName        string    `json:"authorName"`
	//DeleteAvailable   bool      `json:"deleteAvailable"`
	//DownloadAvailable bool      `json:"downloadAvailable"`
	//IsLocked          bool      `json:"isLocked"`
}

func (d *DataRoomFile) DeleteAvailable() bool {
	return false
}

func (d *DataRoomFile) MarshalJSON() ([]byte, error) {
	type Alias DataRoomFile
	return json.Marshal(&struct {
		*Alias
		FileCreated     string `json:"fileCreated"`
		FileExpires     string `json:"fileExpires"`
		DeleteAvailable bool   `json:"deleteAvailable"`
		AuthorName      string `json:"authorName"`
	}{
		Alias:           (*Alias)(d),
		FileCreated:     d.FileCreated.Format(time.RFC3339),
		FileExpires:     d.FileExpires.Format(time.RFC3339),
		DeleteAvailable: d.DeleteAvailable(),
		AuthorName:      "admin",
	})
}

const StatusPublished = "Опубликован"
const StatusUploading = "ИнициированаЗагрузка"

// schemas

type fileMetaSchema struct {
	FileMetaCreated bool      `json:"fileMetaCreated"`
	FileMetaID      uuid.UUID `json:"fileMetaID"`
}

// views

func dataRoomFilesList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:

		roomIds, ok := r.URL.Query()["roomID"]
		if !ok || len(roomIds[0]) < 1 {
			http.Error(w, "Url Param 'roomID' is missing", http.StatusBadRequest)
			return
		}
		roomId := uuid.MustParse(roomIds[0])

		room := DataRoom{Id: roomId}
		err := db.Model(&room).WherePK().Select()
		if err != nil {
			panic(err)
		}

		//check folder
		folderIds, ok := r.URL.Query()["folderID"]
		folderId := room.RootFolderID
		if ok && len(folderIds[0]) > 0 {
			folderId = uuid.MustParse(folderIds[0])

			if room.RootFolderID != folderId {
				http.Error(w, "folder id not in provided room", http.StatusBadRequest)
				return
			}
		}

		files := make([]DataRoomFile, 0)
		err = db.Model(&files).
			Relation("Room").
			Where("folder_id = ?", folderId).
			Select()
		if err != nil {
			panic(err)
		}

		// wrap content with contentData tag
		wrapped := map[string]interface{}{
			"contentData": files,
		}
		res, err := json.Marshal(wrapped)
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
		fmt.Println("Get files from room")

	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}

func dataRoomFilesUpload(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		//fileMetaIds, ok := r.URL.Query()["roomID"]
		//if !ok || len(fileMetaIds[0]) < 1 {
		//	http.Error(w, "Url Param 'fileMetaId' is missing", http.StatusBadRequest)
		//}
		//fileMetaId := uuid.MustParse(fileMetaIds[0])
		//
		//chunkNumbers, ok := r.URL.Query()["roomID"]
		//if !ok || len(chunkNumbers[0]) < 1 {
		//	http.Error(w, "Url Param 'chunkNumber' is missing", http.StatusBadRequest)
		//}
		//chunkNumber := uuid.MustParse(chunkNumbers[0])

		//chunkHashes, ok := r.URL.Query()["roomID"]
		//has_hash := false
		//if ok && len(chunkHashes[0]) > 0 {
		//	has_hash = true
		//	chunkHash := uuid.MustParse(chunkHashes[0])
		//}

		w.WriteHeader(http.StatusNoContent)
		fmt.Println("File upload get")

	case http.MethodPost:
		chunkNumbers, ok := req.URL.Query()["chunkNumber"]
		if !ok || len(chunkNumbers[0]) < 1 {
			http.Error(w, "Url Param 'chunkNumber' is missing", http.StatusBadRequest)
		}
		chunkNumber, err := strconv.Atoi(chunkNumbers[0])
		if err != nil {
			panic(err)
		}

		if chunkNumber == 0 {
			file := new(DataRoomFile)
			room := new(DataRoom)

			fileIds, ok := req.URL.Query()["fileID"]
			if !ok || len(fileIds[0]) < 1 {
				http.Error(w, "Url Param 'fileID' is missing", http.StatusBadRequest)
				return
			}
			fileId := fileIds[0]

			roomIds, ok := req.URL.Query()["roomID"]
			if !ok || len(roomIds[0]) < 1 {
				http.Error(w, "Url Param 'roomID' is missing", http.StatusBadRequest)
				return
			}
			roomId := uuid.MustParse(roomIds[0])

			totalChunksParam, ok := req.URL.Query()["totalChunks"]
			if !ok || len(totalChunksParam[0]) < 1 {
				http.Error(w, "Url Param 'totalChunks' is missing", http.StatusBadRequest)
				return
			}
			totalChunks, err := strconv.Atoi(totalChunksParam[0])
			if err != nil {
				panic(err)
			}

			fileNames, ok := req.URL.Query()["fileName"]
			if !ok || len(fileNames[0]) < 1 {
				http.Error(w, "Url Param 'totalChunks' is missing", http.StatusBadRequest)
				return
			}
			fileName := fileNames[0]
			if err != nil {
				panic(err)
			}

			totalSizeParam, ok := req.URL.Query()["totalSize"]
			if !ok || len(totalSizeParam[0]) < 1 {
				http.Error(w, "Url Param 'totalSize' is missing", http.StatusBadRequest)
				return
			}
			totalSize, err := strconv.Atoi(totalSizeParam[0])
			if err != nil {
				panic(err)
			}

			//check room
			err = db.Model(room).
				Where("id = ?", roomId).
				First()
			if err != nil {
				http.Error(w, "Room with this ID doesn't exist", http.StatusBadRequest)
				return
			}

			//check folder
			folderIds, ok := req.URL.Query()["folderID"]
			folderId := room.RootFolderID
			if ok && len(folderIds[0]) > 0 {
				folderId = uuid.MustParse(folderIds[0])

				if room.RootFolderID != folderId {
					http.Error(w, "folder id not in provided room", http.StatusBadRequest)
					return
				}
			}

			// try to find file
			err = db.Model(file).
				Where("file_id = ?", fileId).
				Where("folder_id = ?", folderId).
				First()

			var res_obj fileMetaSchema
			// file not found - create new
			if err != nil {
				file.FileMetaId = uuid.New()
				file.FileId = fileId
				file.FileName = fileName
				file.FolderId = folderId
				file.FileSize = totalSize
				file.FileStatus = StatusUploading
				file.Chunks = totalChunks
				file.FileCreated = time.Now()
				file.FileExpires = time.Now().AddDate(0, 0, 10)

				_, err = db.Model(file).Insert()
				if err != nil {
					panic(err)
				}

				res_obj.FileMetaCreated = true
			}
			res_obj.FileMetaID = file.FileMetaId

			res, err := json.Marshal(res_obj)
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(res)
		} else {
			//	upload file chunks
			file := new(DataRoomFile)

			fileMetaIds, ok := req.URL.Query()["fileMetaID"]
			if !ok || len(fileMetaIds[0]) < 1 {
				http.Error(w, "Url Param 'fileMetaID' is missing", http.StatusBadRequest)
				return
			}
			fileMetaId := uuid.MustParse(fileMetaIds[0])

			// try to find file
			err = db.Model(file).
				Where("id = ?", fileMetaId).
				First()

			if err != nil {
				http.Error(w,
					"Error getting file metadata parameters! You may not have access rights to the file.",
					http.StatusBadRequest)
				return
			}

			reader, err := req.MultipartReader()
			if err != nil {
				log.Fatal(err)
			}

			file_prefix := fileMetaId.String() + "-"
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
					part.Close()
				} else if err != nil {
					break
					part.Close()
				}

				if part.FormName() != "file" || part.FileName() == "" {
					continue
					part.Close()
				}

				d, err := os.Create("tmp/" + file_prefix + strconv.Itoa(chunkNumber))
				if err != nil {
					log.Fatal(err)
				}
				io.Copy(d, part)
				d.Close()
			}

			// all chunks are uploaded
			if file.Chunks == countFilesByPrefix(file_prefix) {
				destFile, err := os.Create("files/" + fileMetaId.String())
				if err != nil {
					log.Fatal(err)
				}
				defer destFile.Close()

				for i := 1; i <= file.Chunks; i++ {
					partName := "tmp/" + file_prefix + strconv.Itoa(i)
					part, err := os.Open(partName)
					if err != nil {
						log.Fatal(err)
					}

					_, err = io.Copy(destFile, part)
					if err != nil {
						log.Fatal(err)
					}

					err = part.Close()
					if err != nil {
						log.Fatal(err)
					}

					err = os.Remove(partName)
					if err != nil {
						log.Fatal(err)
					}
				}
				// update status
				file.FileStatus = StatusPublished
				_, err = db.Model(file).WherePK().Update()
				if err != nil {
					panic(err)
				}
			}

		}
	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}

func dataRoomFilesDownloadByTicket(w http.ResponseWriter, r *http.Request) {
}

func dataRoomFilesDownloadByMeta(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		fileIds, ok := req.URL.Query()["fileMetaID"]
		if !ok || len(fileIds[0]) < 1 {
			http.Error(w, "Url Param 'fileMetaID' is missing", http.StatusBadRequest)
			return
		}
		fileId := uuid.MustParse(fileIds[0])

		file := new(DataRoomFile)
		// try to find file
		err := db.Model(file).
			Where("id = ?", fileId).
			First()
		if err != nil {
			http.Error(w,
				"Error getting file metadata parameters! You may not have access rights to the file.",
				http.StatusBadRequest)
			return
		}

		if file.FileStatus != StatusPublished {
			http.Error(w,
				"File unavailable",
				http.StatusNotFound)
			return
		}

		readedFile, err := os.Open("files/" + file.FileMetaId.String())
		if err != nil {
			log.Fatal(err)
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.FileName))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Transfer-Encoding", "binary")

		_, err = io.Copy(w, readedFile)
		if err != nil {
			log.Fatal(err)
		}

	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}

func countFilesByPrefix(prefix string) int {
	files, err := ioutil.ReadDir("./tmp")
	if err != nil {
		log.Fatal(err)
	}

	var count int
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) {
			count++
		}
	}
	return count
}
