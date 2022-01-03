# sqlparams for Golang

## Install

```
go get github.com/kaibox-git/sqlparams
```

## Usage

The only method Inline(query, params...) returns sql query with inline parameters, so you can execute it in the database console or log it. Supports pointer and [sql.Null...](https://pkg.go.dev/database/sql#NullBool) parameter types.

MySQL example:
```go
query := `SELECT name FROM table WHERE code=? AND prefix=?`
code := 5
prefix := `some`
sql := sqlparams.Inline(query, code, prefix)
println(sql)
```
Output:
```sql
SELECT name FROM table WHERE code=5 AND prefix='some'
```

PostgreSQL example:
```go
query := `SELECT name FROM table WHERE code=$1 AND prefix=$2`
code := 5
prefix := `some`
params := []interface{}{code, prefix}
sql := sqlparams.Inline(query, params...)
println(sql)
```
Output:
```sql
SELECT name FROM table WHERE code=5 AND prefix='some'
```
Named parameters are supported:
```go
query := `SELECT name FROM table WHERE code=:code AND prefix=:prefix`
m := map[string]interface{}{
        `code`: 5,
        `prefix`: `some`,
}
sql := sqlparams.Inline(query, m)
```
Struct (could be a pointer) is supported:
```go
query := `SELECT name FROM table WHERE code=:code AND prefix=:prefix`
p := struct{
        Code int
        Prefix string
}{
        Code: 5,
        Prefix: `some`,
}
sql := sqlparams.Inline(query, &p)
```
It takes into account the tag 'db' of the struct if using [sqlx](https://github.com/jmoiron/sqlx):
```go
query := `SELECT name FROM table WHERE code=:code AND dep_id=:dep_id`
p := struct{
        Code int
        DepId int `db:"dep_id"`
}{
        Code: 5,
        DepId: 2,
}
sql := sqlparams.Inline(query, p)
```
See more cases in [sqlmaker_test.go](https://github.com/kaibox-git/sqlparams/blob/main/sqlparams_test.go).
