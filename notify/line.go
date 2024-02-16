package notify

import (
	"fmt"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

func SendLineNotify(msg string, lvl int, token string, to string) {
	_, err := messaging_api.NewMessagingApiAPI(
		token,
	)
	if err != nil {
		// Invalid line bot token
		fmt.Println("[x] Invalid linebot token.")
		return
	}

}
