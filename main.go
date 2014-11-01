package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
//	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
)

const version = "0.0.0"

var dbtypes = map[string]bool{
	"sqlite":   true,
	"postgres": true,
}

var database string
var user string
var pass string
var port uint
var dbtype string

func init() {
	const (
		dbtypeDefault   = ""
		dbtypeUsage     = "database type (sqlite or postgres)"
		databaseDefault = ""
		databaseUsage   = "database"
		userDefault     = ""
		userUsage       = "database user"
		passDefault     = ""
		passUsage       = "database password"
		portDefault     = 0
		portUsage       = "http serve port"
	)
	flag.StringVar(&dbtype, "type", dbtypeDefault, dbtypeUsage)
	flag.StringVar(&dbtype, "t", dbtypeDefault, dbtypeUsage+" (shorthand)")
	flag.StringVar(&database, "database", databaseDefault, databaseUsage)
	flag.StringVar(&database, "d", databaseDefault, databaseUsage+" (shorthand)")
	flag.StringVar(&user, "user", userDefault, userUsage)
	flag.StringVar(&user, "u", userDefault, userUsage+" (shorthand)")
	flag.StringVar(&pass, "password", passDefault, passUsage)
	flag.StringVar(&pass, "pass", passDefault, passUsage+" (shorthand)")
	flag.UintVar(&port, "port", portDefault, portUsage)
	flag.UintVar(&port, "p", portDefault, portUsage+" (shorthand)")
}

func printUsage() {
	exeName := os.Args[0]
	fmt.Println(exeName + " " + version + " usage: ")
	fmt.Println("\t" + exeName + " -t postgres -d star-database -u database-user-name -pass database-user-password -p serve-port")
	fmt.Println("\t" + exeName + " -t sqlite -d star-database.sqlite -p serve-port")
	fmt.Println("flags:")
	flag.PrintDefaults()
	fmt.Println("example:\n\t" + exeName + " -d hyg -u jimbob -p nascarrulez -p 8008")
}

type Star struct {
	Id                int64   `json:"id"`
	Name              string  `json:"name"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Z                 float64 `json:"z"`
	Color             float32 `json:"color"`
	AbsoluteMagnitude float32 `json:"absolute-magnitude"`
	Spectrum          string  `json:"spectrum"`
}

func (star *Star) Json() []byte {
	bytes, err := json.Marshal(star)
	if err != nil {
		return nil ///< @todo fix to return JSON error
	}
	return bytes
}

type NullStar struct {
	Id                sql.NullInt64
	Name              sql.NullString
	X                 sql.NullFloat64
	Y                 sql.NullFloat64
	Z                 sql.NullFloat64
	Color             sql.NullFloat64
	AbsoluteMagnitude sql.NullFloat64
	Spectrum          sql.NullString
}

/// returns a Star, with zero values for any null values in the NullStar
func (nstar *NullStar) Star() Star {
	var star Star
	if nstar.Id.Valid {
		star.Id = nstar.Id.Int64
	}
	if nstar.Name.Valid {
		star.Name = nstar.Name.String
	}
	if nstar.X.Valid {
		star.X = nstar.X.Float64
	}
	if nstar.Y.Valid {
		star.Y = nstar.Y.Float64
	}
	if nstar.Z.Valid {
		star.Z = nstar.Z.Float64
	}
	if nstar.Color.Valid {
		star.Color = float32(nstar.Color.Float64)
	}
	if nstar.AbsoluteMagnitude.Valid {
		star.AbsoluteMagnitude = float32(nstar.AbsoluteMagnitude.Float64)
	}
	if nstar.Spectrum.Valid {
		star.Spectrum = nstar.Spectrum.String
	}
	return star
}

func postgresDbManager(user string, pass string, getStar chan struct {
	id       int64
	callback chan Star
}) {
	//	fmt.Println("Opening " + "postgres://" + user + ":" + pass + "@localhost/"+database)

	//	url := "postgres://"+user+":"+pass+"@localhost/"+database
	//	connection, _ := pq.ParseURL(url)
	//	connection += " sslmode=require"
	connstr := "user=" + user + " dbname=" + database + " password=" + pass
	fmt.Println("Opening " + connstr)
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		fmt.Println("postgresDbManager FAILED open")
		log.Fatal(err)
	}
	defer db.Close()

	sql := "select propername, x, y, z, colorindex, absmag, spectrum from hygxyz where starid = $1"
	stmt, err := db.Prepare(sql)
	if err != nil {
		fmt.Println("dbManager FAILED prepare")
		log.Fatal(err)
	}

	for {
		request := <-getStar

		rows, err := stmt.Query(request.id)

		if err != nil {
			fmt.Println("dbManager FAILED query")
			log.Fatal(err)
		}

		if !rows.Next() {
			request.callback <- Star{}
			continue
		}

		var nstar NullStar
		err = rows.Scan(&nstar.Name, &nstar.X, &nstar.Y, &nstar.Z, &nstar.Color, &nstar.AbsoluteMagnitude, &nstar.Spectrum)
		if err != nil {
			fmt.Println("dbManager FAILED scan")
			log.Fatal(err)
		}
		star := nstar.Star()
		star.Id = request.id
		request.callback <- star // return zero values for null fields

		rows.Close()
	}
}

func sqliteDbManager(getStar chan struct {
	id       int64
	callback chan Star
}) {

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		fmt.Println("sqliteDbManager FAILED open")
		log.Fatal(err)
	}
	defer db.Close()

	sql := "select propername, x, y, z, colorindex, absmag, spectrum from hygxyz where starid = ?"
	stmt, err := db.Prepare(sql)
	if err != nil {
		fmt.Println("dbManager FAILED prepare")
		log.Fatal(err)
	}

	for {
		request := <-getStar

		rows, err := stmt.Query(request.id)

		if err != nil {
			fmt.Println("dbManager FAILED query")
			log.Fatal(err)
		}

		if !rows.Next() {
			request.callback <- Star{}
			continue
		}

		var nstar NullStar
		err = rows.Scan(&nstar.Name, &nstar.X, &nstar.Y, &nstar.Z, &nstar.Color, &nstar.AbsoluteMagnitude, &nstar.Spectrum)
		if err != nil {
			fmt.Println("dbManager FAILED scan")
			log.Fatal(err)
		}
		star := nstar.Star()
		star.Id = request.id
		request.callback <- star // return zero values for null fields

		rows.Close()
	}
}

const services = `{"services" : ["star/{id}"]}`

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	if !dbtypes[dbtype] || database == "" || port == 0 || port > 65535 {
		printUsage()
		return
	}
	if dbtype == "postgres" {
		if user == "" || pass == "" {
			printUsage()
			return
		}
	}
	if dbtype == "sqlite" {
		_, err := os.Stat(database)
		if os.IsNotExist(err) {
			fmt.Printf("sqlite database file does not exist: %s", database)
			return
		}
	}

	getStar := make(chan struct {
		id       int64
		callback chan Star
	}, 100)

	if dbtype == "postgres" {
		go postgresDbManager(user, pass, getStar)
	} else {
		go sqliteDbManager(getStar)
	}

	serveServicesList := func(w http.ResponseWriter) error {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(services)))
		_, err := w.Write([]byte(services))
		return err
	}

	serveStar := func(w http.ResponseWriter, id int64) error {
		getStarCallback := make(chan Star)
		getStar <- struct {
			id       int64
			callback chan Star
		}{id, getStarCallback}
		star := <-getStarCallback

		starjson := star.Json()

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Content-Length", strconv.Itoa(len(starjson)))

		_, err := w.Write(starjson)
		return err
	}

	http.HandleFunc("/star/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Serving request to " + r.URL.Path)

		staridStr := r.URL.Path[len("/star/"):]
		starid, err := strconv.ParseInt(staridStr, 10, 64)
		_, err = strconv.Atoi(staridStr)

		if err != nil {
			serveServicesList(w)
			return
		}
		err = serveStar(w, starid)
		if err != nil {
			fmt.Println(err)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := serveServicesList(w)
		if err != nil {
			fmt.Println(err)
		}
	})

	fmt.Println("Serving on " + strconv.Itoa(int(port)) + "...")
	http.ListenAndServe(":"+strconv.Itoa(int(port)), nil)
}
