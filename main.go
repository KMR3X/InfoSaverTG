package main

import (
	//"fmt"
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gocql "github.com/gocql/gocql"
	gocqlx "github.com/scylladb/gocqlx"

	qb "github.com/scylladb/gocqlx/qb"
	table "github.com/scylladb/gocqlx/table"
	zap "go.uber.org/zap"
)

type UserExistence struct {
	ID int64 `json:"id"`
}

type UserInfoFS struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot,omitempty"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	UserName     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

type query struct {
	stmt  string
	names []string
}

type statements struct {
	del query
	ins query
	sel query
}

type Record struct {
	ID           string `db:"id"`
	IsBot        string `db:"is_bot"`
	FirstName    string `db:"first_name"`
	LastName     string `db:"last_name"`
	UserName     string `db:"user_name"`
	LanguageCode string `db:"language_code"`
}

func main() {
	//параметры запуска контейнера:
	//docker run --name node1 -p 9042:9042 -d scylladb/scylla --broadcast-address 127.0.0.1 --listen-address 0.0.0.0 --broadcast-rpc-address 127.0.0.1

	logger := zap.NewExample()

	//инициализация кластера, сессии и подключение к бд
	cluster := CreateCluster(gocql.Quorum, "is_3000", "127.0.0.1")
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		logger.Fatal("Ошибка подключения", zap.Error(err))
	}
	defer session.Close()

	//инициализация бота
	bot, err := tgbotapi.NewBotAPI("BOT_TOKEN")
	if err != nil {
		log.Fatal(err)
	}

	//Информация о владельце
	log.Printf("Запуск бота %s", bot.Self.UserName)

	//получение обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	//канал для обновлений
	updates := bot.GetUpdatesChan(u)

	var msgText string

	//работа с сообщениями из обновлений
	for update := range updates {
		if update.Message == nil {
			continue
		}

		//Вывод в лог отправленного пользователем сообщения
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		//Проверка на наличие пользователя в бд, если нет - сохранение его данных
		if selectQuery(session, logger, int(update.Message.From.ID)) {
			msgText = "Такой пользователь уже существует. "
		} else {
			SaveInfoDB(session, logger, update.Message)
			msgText = "Информация успешно сохранена! "
		}

		//Отправка сообщения в тот же чат
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msgText+update.Message.From.UserName))
	}
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

// Запрос на вставку
func insertQuery(session *gocql.Session, id, isbot, firstname, lastName, username, languagecode string, logger *zap.Logger) {
	logger.Info("Вставка " + id + "......")

	r := Record{
		ID:           id,
		IsBot:        isbot,
		FirstName:    firstname,
		LastName:     lastName,
		UserName:     username,
		LanguageCode: languagecode,
	}

	err := gocqlx.Query(session.Query(stmts.ins.stmt), stmts.ins.names).BindStruct(r).ExecRelease()
	if err != nil {
		logger.Error("insert is_3000.users", zap.Error(err))
	}
}

// Запрос показа данных
func selectQuery(session *gocql.Session, logger *zap.Logger, id int) bool {
	//срез, в который будут считаны данные
	var rs []Record

	//считывание данных в rs
	err := gocqlx.Query(session.Query(stmts.sel.stmt), stmts.sel.names).Select(&rs)
	if err != nil {
		logger.Warn("select is_3000.users", zap.Error(err))
		return false
	}

	//поиск такого же пользователя в бд по ID
	for _, r := range rs {
		if r.ID == strconv.Itoa(id) {
			return true
		}
	}
	return false
}

// Сохранение информации о сообщении в БД
func SaveInfoDB(session *gocql.Session, logger *zap.Logger, message *tgbotapi.Message) {
	//Приведение форматов данных
	strID := strconv.Itoa(int(message.From.ID))
	strIsBot := strconv.FormatBool(message.From.IsBot)

	//Запрос на вставку
	insertQuery(session, strID, strIsBot, message.From.FirstName,
		message.From.LastName, message.From.UserName, message.From.LanguageCode, logger)
}
