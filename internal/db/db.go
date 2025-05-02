package db

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int64
	Name           string
	Password       string
	OriginPassword string
}

var (
	ctx context.Context
	db  *sql.DB
)

func (u User) ComparePassword(u2 User) error {
	err := compare(u2.Password, u.OriginPassword)
	if err != nil {
		log.Println("DB: Auth fail")
		return err
	}
	log.Println("DB: Auth success")
	return nil
}

func createTables(ctx context.Context, db *sql.DB) error {
	const userTable = `
	CREATE TABLE IF NOT EXISTS users(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		password TEXT
	);
	`

	if _, err := db.ExecContext(ctx, userTable); err != nil {
		return err
	}

	// const expressionTable = `
	// CREATE TABLE IF NOT EXISTS expressions(
	// 	id INTEGER PRIMARY KEY AUTOINCREMENT,
	// 	uid INTEGER,
	// 	status TEXT,
	// 	RootNodeID INTEGER,
	// 	Result TEXT
	// );
	// `

	// if _, err := db.ExecContext(ctx, expressionTable); err != nil {
	// 	return err
	// }

	return nil
}

func InsertUser(user *User) (int64, error) {
	var q = `
	INSERT INTO users (name, password) values ($1, $2)
	`

	result, err := db.ExecContext(ctx, q, user.Name, user.Password)
	if err != nil {
		return 0, nil
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func SelectUser(name string) (User, error) {
	var (
		user User
		err  error
	)

	var q = "SELECT id, name, password FROM users WHERE name=$1"
	err = db.QueryRowContext(ctx, q, name).Scan(&user.ID, &user.Name, &user.Password)
	return user, err
}

func GenerateHash(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hash := string(hashedBytes[:])
	return hash, nil
}

func compare(hash string, s string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}

func Init() error {
	ctx = context.TODO()
	var err error

	db, err = sql.Open("sqlite3", "data/store.db")
	if err != nil {
		return err
	}

	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	if err = createTables(ctx, db); err != nil {
		return err
	}

	return nil
}

func Stop() {
	db.Close()
}

// func main() {
// 	ctx := context.TODO()

// 	db, err := sql.Open("sqlite3", "store.db")
// 	if err != nil {
// 		panic(err)
// 	}

// 	defer db.Close()

// 	err = db.PingContext(ctx)
// 	if err != nil {
// 		panic(err)
// 	}

// 	if err = createTable(ctx, db); err != nil {
// 		panic(err)
// 	}

// 	password, err := generate("qwertyqwerty")
// 	if err != nil {
// 		panic(err)
// 	}

// 	user := &User{
// 		Name:           "New Name",
// 		Password:       password,
// 		OriginPassword: "qwertyqwerty",
// 	}
// 	userID, err := insertUser(ctx, db, user)
// 	if err != nil {
// 		log.Println("user already exists")
// 	} else {
// 		user.ID = userID
// 	}

// 	userFromDB, err := selectUser(ctx, db, user.Name)
// 	if err != nil {
// 		panic(err)
// 	}

// 	user.ComparePassword(userFromDB)
// 	user.Password, err = generate("fail passwd")
// 	if err != nil {
// 		panic(err)
// 	}
// 	user.OriginPassword = "fail passwd"
// 	user.ComparePassword(userFromDB)
// }
