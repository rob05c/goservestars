package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
)

const version = "0.0.0"

var database string
var user string
var pass string
var port uint

func init() {
	const (
		databaseDefault = ""
		databaseUsage   = "database"
		userDefault     = ""
		userUsage       = "database user"
		passDefault     = ""
		passUsage       = "database password"
		portDefault     = 0
		portUsage       = "http serve port"
	)
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
	fmt.Println("\t" + exeName + " -d star-database -u database-user-name -pass database-user-password -p serve-port")
	fmt.Println("flags:")
	flag.PrintDefaults()
	fmt.Println("example:\n\t" + exeName + " -d hyg -u jimbob -p nascarrulez -p 8008")
}

type Star struct {
	Id                int64
	Name              string
	X                 float64
	Y                 float64
	Z                 float64
	Color             float32
	AbsoluteMagnitude float32
	Spectrum          string
}

func (star *Star) Json() string {
	return "{" +
		"\"id\": " + strconv.Itoa(int(star.Id)) + ", " +
		"\"name\": \"" + star.Name + "\", " +
		"\"x\": " + strconv.FormatFloat(star.X, 'f', 15, 32) + ", " +
		"\"y\": " + strconv.FormatFloat(star.Y, 'f', 15, 32) + ", " +
		"\"z\": " + strconv.FormatFloat(star.Z, 'f', 15, 32) + ", " +
		"\"color\": " + strconv.FormatFloat(float64(star.Color), 'f', 15, 32) + ", " +
		"\"absolute-magnitude\": " + strconv.FormatFloat(float64(star.AbsoluteMagnitude), 'f', 15, 32) + ", " +
		"\"spectrum\": \"" + star.Spectrum + "\"}"
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

func dbManager(user string, pass string, getStar chan struct {
	id       int64
	callback chan Star
}) {
	fmt.Println("Opening " + "postgres://" + user + ":" + pass + "@localhost/")
	db, err := sql.Open("postgres", "postgres://"+user+":"+pass+"@localhost/"+database)
	if err != nil {
		log.Fatal(err)
	}

	sql := "select propername, x, y, z, colorindex, absmag, spectrum from hygxyz where starid = $1"
	stmt, err := db.Prepare(sql)
	if err != nil {
		log.Fatal(err)
	}

	for {
		request := <-getStar

		rows, err := stmt.Query(request.id)

		if err != nil {
			log.Fatal(err)
		}

		if !rows.Next() {
			request.callback <- Star{}
			continue
		}

		var nstar NullStar
		err = rows.Scan(&nstar.Name, &nstar.X, &nstar.Y, &nstar.Z, &nstar.Color, &nstar.AbsoluteMagnitude, &nstar.Spectrum)
		if err != nil {
			log.Fatal(err)
		}
		star := nstar.Star()
		star.Id = request.id
		request.callback <- star // return zero values for null fields

		rows.Close()
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	if database == "" || user == "" || pass == "" || port == 0 || port > 65535 {
		printUsage()
		return
	}

	getStar := make(chan struct {
		id       int64
		callback chan Star
	}, 100)

	go dbManager(user, pass, getStar)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		getStarCallback := make(chan Star)

		fmt.Println("Serving request to " + r.URL.Path)
		if r.URL.Path[:len("/star")] == "/star" && len(r.URL.Path) > len("/star")+1 {
			staridStr := r.URL.Path[len("/star/"):]
			starid, err := strconv.Atoi(staridStr)
			if err != nil {
				fmt.Println("ignoring request for non-integral star id")
				return
			}

			getStar <- struct {
				id       int64
				callback chan Star
			}{int64(starid), getStarCallback}
			star := <-getStarCallback
			fmt.Fprintf(w, star.Json())

		} else {
			fmt.Println("ignoring unknown request")
			return
		}

	})

	fmt.Println("Serving on " + strconv.Itoa(int(port)) + "...")
	http.ListenAndServe(":"+strconv.Itoa(int(port)), nil)
}
