package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func main() {
	//инициализация бота
	bot, err := tgbotapi.NewBotAPI("6904018802:AAHC2WDBPW4za575perboin2mr1LNKq2ZV4")
	if err != nil {
		log.Fatal(err)
	}

	//bot.Debug = true

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

		//Сохранение информации
		if CheckUser(update.Message) == false {
			SaveInfo(update.Message)
			msgText = "Информация успешно сохранена! "
		} else {
			msgText = "Такой пользователь уже существует. "
		}

		//Отправка сообщения в тот же чат
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msgText+update.Message.From.UserName))
	}
}

func CheckUser(message *tgbotapi.Message) bool {
	//var userIDslice []int64

	//Открытие и отложенное закрытие файла
	saveFile, err := os.Open("savedInfo.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer saveFile.Close()

	//Сканнер для чтения и объект типа UserExistence
	sc := bufio.NewScanner(saveFile)
	var ue UserExistence

	//Чтение файла, выделение поля ID, сверка с ID отправившего сообщение
	for sc.Scan() {
		//fmt.Printf("%s\n", sc.Text())
		data := []byte(sc.Text())

		//пустой файл
		if len(data) == 0 {
			return false
		}

		//конвертация из json в string, присваивание ID прочитанного в созданный объект
		if err := json.Unmarshal(data, &ue); err != nil {
			log.Fatal(err)
		}

		//если такой пользователь уже записан, то заново не создается
		if ue.ID == message.From.ID {
			return true
		}
	}
	//если не найдено такого же пользователя, то переходим к записи его информации в файл
	return false
}

// Сохранение всей информации о сообщении в .txt файл
func SaveInfo(message *tgbotapi.Message) {

	//Структура сохраняемых данных, инициализация объекта
	currUser := UserInfoFS{
		ID:           message.From.ID,
		IsBot:        message.From.IsBot,
		FirstName:    message.From.FirstName,
		LastName:     message.From.LastName,
		UserName:     message.From.UserName,
		LanguageCode: message.From.LanguageCode,
	}

	//конвертация в формат json
	info, err := json.Marshal(currUser)
	if err != nil {
		log.Fatal(err)
	}

	//Открытие/создание файла для сохранения, файл может быть только дополнен, не переписан
	saveFile, err := os.OpenFile("savedInfo.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	//запись информации в файл с добавлением переноса строки
	if _, err := saveFile.WriteString(string(info) + "\n"); err != nil {
		log.Fatal(err)
	}

	//закрытие файла
	saveFile.Close()
}
