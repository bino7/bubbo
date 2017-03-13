package bubbo

import (
	"github.com/asdine/storm"
)

var (
	store_path = "bubbo.db"
	db *storm.DB
)

func initStore() {
	d, err := storm.Open(store_path)
	if err != nil {
		panic(err)
	}
	db = d

	if err := db.Init(&User{}); err != nil {
		panic(err)
	}

	if err := db.Init(&Media{}); err != nil {
		panic(err)
	}
}