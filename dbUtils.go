package main

import (
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"log"
)

func NewDBConn() (con *pg.DB) {
	address := fmt.Sprintf("%s:%s", "localhost", "5432")
	options := &pg.Options{
		User:     "postgres",
		Password: "root",
		Addr:     address,
		Database: "datarooms",
		PoolSize: 50,
	}
	con = pg.Connect(options)
	if con == nil {
		log.Fatal("cannot connect to postgres")
	}
	return
}

func createSchemas(db *pg.DB) error {
	models := []interface{}{
		(*DataRoom)(nil),
		(*DataRoomFile)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp: false,
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}