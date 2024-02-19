package main

import (
	"log"
	"strconv"

	config "github.com/KMR3X/InfoSaverTG/config"
	database "github.com/KMR3X/InfoSaverTG/internal"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	//параметры запуска контейнера:
	//docker run --name node1 -p 9042:9042 -d scylladb/scylla --broadcast-address 127.0.0.1 --listen-address 0.0.0.0 --broadcast-rpc-address 127.0.0.1

	session := database.ConnectDB()
	defer session.Close()

	//инициализация бота
	bot, err := tgbotapi.NewBotAPI(config.BotInfo("token"))
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

		var userEx = database.Record{
			ID:           strconv.Itoa(int(update.Message.From.ID)),
			IsBot:        strconv.FormatBool(update.Message.From.IsBot),
			FirstName:    update.Message.From.FirstName,
			LastName:     update.Message.From.LastName,
			UserName:     update.Message.From.UserName,
			LanguageCode: update.Message.From.LanguageCode,
		}

		//Проверка на наличие пользователя в бд, если нет - сохранение его данных
		if database.SelectQuery(session, userEx.ID) {
			msgText = "Такой пользователь уже существует. "
		} else {
			database.SaveInfoDB(session, userEx)
			msgText = "Информация успешно сохранена! "
		}

		//Отправка сообщения в тот же чат
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msgText+update.Message.From.UserName))
	}
}
