# A Rest API application using Golang on Mongo database storage

This application highlights the development of a simple Rest API using _golang_:

* It handles _GET(all), GET, POST, DELETE_ on Person objects (_go struct_)
* All data are saved on Mongo (_db=people, collection=people_)
* On application startup, 2 persons are dumped on Mongo to initialize / tests
* The application also handles the CORS config via an _http.Handler_ wrapper

## External dependencies

* [mgo.v2](gopkg.in/mgo.v2) golang mongo driver
* [mgo.v2/bson](gopkg.in/mgo.v2/bson) mongo serdes go routines
* [mux](github.com/gorilla/mux) HTTP router

you can get this dependencies via `go get` package utility

## Application compilation and startup
Compile application (`go build`) and then launch using

```
  ./rest-mongo <http port> <mongo url>
```

_example_

```
  ./rest-mongo 8080 127.0.0.1
```

## API examples using _curl_ (on localhost)
(once the application started, say on port 8080)

### List all persons
```
curl -vX GET http://localhost:8080/people
```

### Add a new person
```
curl -vX POST http://localhost:8080/people -d '{"firstname": "DOE", "lastname": "John"}'
```
this returns the _Location_ to the resource

### Get a single person
```
curl -vX GET http://localhost:8080/people/{resource_id}
```

### Delete a single person
```
curl -vX DELETE http://localhost:8080/people/{resource_id}
```


