package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/coopernurse/gorp"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFormat   = "2006-01-02 15:04:05"
	jsonFormat = "2006-01-02"
)

var (
	Dbm *gorp.DbMap
)

type JDate time.Time
type CustomTypeConverter struct{}

func init() {
	Dbm = newDbMap()
	Dbm.TypeConverter = CustomTypeConverter{}

	Dbm.AddTableWithName(Tweet{}, "tweets").SetKeys(true, "Id")

	//Dbm.TraceOn("[gorp]", r.INFO)
	err := Dbm.CreateTablesIfNotExists()
	checkErr(err, "create tables failed")

}

func newDbMap() *gorp.DbMap {
	dialect, driver := dialectAndDriver()
	return &gorp.DbMap{Db: connect(driver), Dialect: dialect}
}

func dialectAndDriver() (gorp.Dialect, string) {
	switch os.Getenv("ENV") {
	case "PRODUCTION":
		return gorp.PostgresDialect{}, "postgres"
	default:
		return gorp.SqliteDialect{}, "sqlite3"
	}
}

func connect(driver string) *sql.DB {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		panic("DB_DSN env variable is not set")
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		panic("Error connecting to db: " + err.Error())
	}
	return db
}

func (d JDate) MarshalJSON() ([]byte, error) {
	return json.Marshal((*time.Time)(&d).Format(jsonFormat))
}

func (d *JDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(jsonFormat, s)
	if err != nil {
		return err
	}
	*d = JDate(t)
	return nil
}

func (me CustomTypeConverter) ToDb(val interface{}) (interface{}, error) {
	switch t := val.(type) {
	case JDate:
		return time.Time(t), nil
	}
	return val, nil
}

func (me CustomTypeConverter) FromDb(target interface{}) (gorp.CustomScanner, bool) {
	switch target.(type) {
	case *JDate:
		binder := func(holder, target interface{}) error {
			// time.Time is returned from db as string
			s, ok := holder.(*string)
			if !ok {
				return errors.New("FromDb: Unable to convert JDate to *string")
			}
			st, ok := target.(*JDate)
			if !ok {
				return errors.New(fmt.Sprint("FromDb: Unable to convert target to *JDate: ", reflect.TypeOf(target)))
			}
			t, _ := time.Parse(dbFormat, *s)
			*st = JDate(t)
			return nil
		}
		return gorp.CustomScanner{new(string), target, binder}, true
	}
	return gorp.CustomScanner{}, false
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}
