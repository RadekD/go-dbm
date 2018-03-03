# go-dbm [![Travis-CI](https://travis-ci.org/RadekD/go-dbm.svg)](https://travis-ci.org/RadekD/go-dbm) [![GoDoc](https://godoc.org/github.com/RadekD/go-dbm?status.svg)](https://godoc.org/github.com/RadekD/go-dbm)

Simple CRUD wrapper on *sql.DB with powerfull struct unpacking 

`go get github.com/RadekD/go-dbm`

## Usage
```go
import "github.com/RadekD/go-dbm"

var crud *dbm.CRUD
func main() {
    var err error

    crud, err = dbm.Open("mysql", "root@tcp(127.0.0.1:3306)/test?collation=utf8mb4_unicode_ci&parseTime=true")
    if err != nil {
        log.Fatal("invalid connection")
    }
    var mystruct struct{
        Name string
    }
    err = crud.Select(&mystruct, "SELECT Name FROM test")
    if err != nil {
        log.Println("handle error")
    }
}
```

## Licence

MIT License