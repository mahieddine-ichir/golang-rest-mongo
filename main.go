package main

/* available subpackages
gopkg.in/mgo.v2 (download)
gopkg.in/mgo.v2/internal/json
gopkg.in/mgo.v2/internal/scram
gopkg.in/mgo.v2/bson
gopkg.in/mgo.v2
*/
import (
	"gopkg.in/mgo.v2"
	"encoding/json"
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
	//"strconv"
	"os"
	"os/signal"
	"syscall"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	ID   bson.ObjectId `json:"id" bson:"_id"`
	//ID        string   `json:"id,omitempty"`
	Firstname string   `json:"firstname,omitempty"`
	Lastname  string   `json:"lastname,omitempty"`
	Address   *Address `json:"address,omitempty"`
}

type Address struct {
	City  string `json:"city,omitempty"`
	State string `json:"state,omitempty"`
}

// Mongo db and collection name -> TODO command line arguments
const db = "people"
const collection = "people"

// global variables: mongo session
var (
	session *mgo.Session
)

// shorthand for the panic on errour routine
func PanicError(err error) {
	if err != nil {
		panic(err)
	}
}

// open a mongo session: done on startup
func openSession(url string) {
	fmt.Println("> Open mongo session on", url)
	var err error
	session, err = mgo.Dial(url)
	PanicError(err)
	session.SetMode(mgo.Monotonic, true) // lookup for the other modes
	go func() {
		closeSession()
	}()
}

// close the mongo session: done on application termination
func closeSession() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for range signals {
		fmt.Println("Closing mongo session")
		signal.Stop(signals)
		session.Close()
		return
	}
}

// routines to execute on application startup
func onstart(url string) {

	// open a MongoSession
	openSession(url)

	// insert some test data
	people := make([]Person, 2)
	people[0] = Person{Firstname:"Mahieddine Mehdi", Lastname:"ICHIR", Address: &Address{City:"Algeria", State:"Algiers"}}
	people[1] = Person{Firstname:"Rafika", Lastname:"ICHIR", Address: &Address{City:"France", State:"Paris"}}

	c := session.DB(db).C(collection)
	for _, p := range people {
		if find(p, c) == (Person{}) {
			fmt.Println("Populating ", p.Lastname, p.Firstname)
			p.save(c)
		}
	}
}

// Main application
// Usage ./rest-mongo <http listening port> <mongo connection url>
//
func main()  {

	args := os.Args[1:] // application command line params
	// mongo connection URL
	url := args[1]
	// http port
	port := args[0]

	// startup routine
	onstart(url)

	// Configure HTTP Routes (mux package)
	router := mux.NewRouter().StrictSlash(true)

	// Routes
	router.HandleFunc("/people", GetPeople).Methods("GET")
	router.HandleFunc("/people/{id}", GetOnePeople).Methods("GET")
	router.HandleFunc("/people", AddPeople).Methods("POST")
	router.HandleFunc("/people/{id}", DeletePeople).Methods("DELETE")

	fmt.Println("> Starting http server on port", port)
	http.ListenAndServe(":"+port, cors(router)) // wrapping with CORS config
}

// a log wrapper that can be used
func log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Calling ... ", r.Method, r.URL)
		h.ServeHTTP(w, r)
		fmt.Println("Done ... ", r.Method, r.URL)
	})
}

// CORS config
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		origin := "*"
		if r.Header.Get("Origin") != "" {
			origin = r.Header.Get("Origin")
		}
		w.Header().Set("Access-Control-Allow-Origin", origin);
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH");
		w.Header().Set("Access-Control-Allow-Headers", "accept, authorization, content-type, x-requested-with");
		w.Header().Set("Access-Control-Allow-Credentials", "true");
		w.Header().Set("Access-Control-Max-Age", "1");
		w.Header().Set("Access-Control-Expose-Headers", "Location");

		if (r.Method == "OPTIONS") {
			return
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

// load all Persons on the HTTP Response
// TODO add pagination
func GetPeople(w http.ResponseWriter, r *http.Request) {

	c := session.DB(db).C(collection)
	json.NewEncoder(w).Encode(findAll(c))
}

// load a single Person on the HTTP Response by its id
func GetOnePeople(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			w.WriteHeader(http.StatusNotFound) // if id is not a valid UUID
		}
	}()
	id := mux.Vars(r)["id"]
	c := session.DB(db).C(collection)
	p := findById(id, c)
	if p == (Person{}) {
		w.WriteHeader(http.StatusNotFound) // id not found
	} else {
		json.NewEncoder(w).Encode(p)
	}
}

// Add a new Person
func AddPeople(w http.ResponseWriter, r *http.Request) {
	p := Person{}
	json.NewDecoder(r.Body).Decode(&p)
	c := session.DB(db).C(collection)
	if find(p, c) != (Person{}) {
		w.WriteHeader(http.StatusFound) // already exists
	} else {
		p.ID = bson.NewObjectId()
		err := c.Insert(&p)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			scheme := "http"
			if r.URL.Scheme != "" {
				scheme = r.URL.Scheme
			}
			w.Header().Set("Location", scheme+"://"+r.Host+r.URL.Path+"/"+p.ID.Hex())
			w.WriteHeader(http.StatusCreated)
		}
	}
}

// Delete a Person by it's ID
func DeletePeople(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			w.WriteHeader(http.StatusNotFound) // if id is not a valid UUID
		}
	}()
	vars := mux.Vars(r)
	c := session.DB(db).C(collection)
	p := findById(vars["id"], c)
	if p == (Person{}) {
		w.WriteHeader(http.StatusNotFound) // id not found
	} else {
		if delete(p, c) != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// Mongo routines
func (p Person) save(c *mgo.Collection) (bson.ObjectId, error) {

	// FIXME why these different methods are not working
	//info, err := c.Upsert(nil, &p)
	//return info.UpsertedId.(bson.ObjectId), err
	//_, err := c.UpsertId(p.ID, p)
	//return p.ID, err

	p.ID = bson.NewObjectId()
	err := c.Insert(&p)
	return p.ID, err
}

// Find a Person by its firstname and lastname
// TODO make use of interfaces
func find(p Person, c *mgo.Collection) (result Person) {
	result = Person{}
	c.Find(bson.M{"firstname": p.Firstname, "lastname": p.Lastname}).One(&result)
	return
}

func findById(id string, c *mgo.Collection) Person {
	result := Person{}
	// c.Find(bson.M{"_id": id}).One(&result) // -> FIXME not found
	err := c.FindId(bson.ObjectIdHex(id)).One(&result)
	if err != nil {
		fmt.Println(err)
	}
	return result
}

func findAll(c *mgo.Collection) []Person {
	p := make([]Person, 10)
	c.Find(bson.M{}).All(&p) // TODO set max results
	return p
}

func delete(p Person, c *mgo.Collection) error {
	return c.RemoveId(p.ID)
}