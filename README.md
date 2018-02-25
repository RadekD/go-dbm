# go-dbm [![Travis-CI](https://travis-ci.org/RadekD/go-dbm.svg)](https://travis-ci.org/RadekD/go-dbm) [![GoDoc](https://godoc.org/github.com/RadekD/go-dbm?status.svg)](https://godoc.org/github.com/RadekD/go-dbm)

Dead simple ORM-ish library inspired by Gorp

`go get github.com/RadekD/go-dbm`

## Usage
```go
import "github.com/RadekD/go-dbm"

func main() {
    dbPool, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/test?collation=utf8mb4_unicode_ci&parseTime=true")
    if err != nil {
        log.Fatal("invalid connection")
    }

    db := &dbm.MySQL{
        DB: dbPool,
    }
    
    var mystruct struct{
        Name string
    }
    err = db.Select(&mystruct, "SELECT Name FROM test")
    if err != nil {
        log.Println("handle error")
    }
}
```

## Licence

MIT License