package config

import (
	"io"
	"log"
	"os"
)

type botParams struct {
	name  string
	token string
}

var info = botParams{
	name:  "is_3000",
	token: tokenStr,
}

func BotInfo(param string) string {
	switch param {
	case "name":
		return info.name
	case "token":
		return info.token
	default:
		return "error"
	}
}

var tokenStr = func() string {
	tokenFile, err := os.Open("token.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer tokenFile.Close()

	b, err := io.ReadAll(tokenFile)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}()
