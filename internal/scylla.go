package internal

import (
	"log"
	"time"

	gocql "github.com/gocql/gocql"
	gocqlx "github.com/scylladb/gocqlx"
	qb "github.com/scylladb/gocqlx/qb"
	table "github.com/scylladb/gocqlx/table"
)

type Rec = Record

// Создание кластера бд
func CreateCluster(consistency gocql.Consistency, keyspace string, hosts ...string) *gocql.ClusterConfig {
	retryPolicy := &gocql.ExponentialBackoffRetryPolicy{
		Min:        time.Second,
		Max:        10 * time.Second,
		NumRetries: 5,
	}
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Timeout = 5 * time.Second
	cluster.RetryPolicy = retryPolicy
	cluster.Consistency = consistency
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	return cluster
}

// инициализация кластера, сессии и подключение к бд
func ConnectDB() *gocql.Session {
	cluster := CreateCluster(gocql.Quorum, "is_3000", "127.0.0.1")
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		log.Fatal("Ошибка подключения", err)
	}
	return session
}

var stmts = createStatements()

// создание выражений для работы с бд
func createStatements() *statements {
	//задание схемы таблицы
	m := table.Metadata{
		Name:    "users",
		Columns: []string{"id", "is_bot", "first_name", "last_name", "user_name", "language_code"},
		PartKey: []string{"id"},
	}
	tbl := table.New(m)

	//Методы работы с данными
	deleteStmt, deleteNames := tbl.Delete()
	insertStmt, insertNames := tbl.Insert()
	selectStmt, selectNames := qb.Select(m.Name).Columns(m.Columns...).ToCql()

	return &statements{
		del: query{
			stmt:  deleteStmt,
			names: deleteNames,
		},
		ins: query{
			stmt:  insertStmt,
			names: insertNames,
		},
		sel: query{
			stmt:  selectStmt,
			names: selectNames,
		},
	}
}

// Запрос на вставку
func InsertQuery(session *gocql.Session, user Record) {
	log.Println("Вставка " + user.ID + "......")

	r := Record{
		ID:           user.ID,
		IsBot:        user.IsBot,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		UserName:     user.UserName,
		LanguageCode: user.LanguageCode,
	}

	err := gocqlx.Query(session.Query(stmts.ins.stmt), stmts.ins.names).BindStruct(r).ExecRelease()
	if err != nil {
		log.Fatal("insert is_3000.users", err)
	}
}

// Запрос показа данных
func SelectQuery(session *gocql.Session, id string) bool {
	//срез, в который будут считаны данные
	var rs []Rec

	//считывание данных в rs
	err := gocqlx.Query(session.Query(stmts.sel.stmt), stmts.sel.names).Select(&rs)
	if err != nil {
		log.Println("select is_3000.users", err)
		return false
	}

	//поиск такого же пользователя в бд по ID
	for _, r := range rs {
		if r.ID == id {
			return true
		}
	}
	return false
}

// Сохранение информации о сообщении в БД
func SaveInfoDB(session *gocql.Session, user Record) {
	//Запрос на вставку
	InsertQuery(session, user)
}
