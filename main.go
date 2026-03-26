package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/jibber_jabber"
	maxigobot "github.com/maxigo-bot/maxigo-bot"
	maxigo "github.com/maxigo-bot/maxigo-client"
	"github.com/xlab/closer"
)

func main() {
	mainCtx, mainCancel = context.WithCancel(context.Background())
	ttCtx, ttCancel = context.WithCancel(mainCtx)

	var (
		err error
	)
	defer closer.Close()

	closer.Bind(func() {
		if err != nil {
			let.Println(err)
			SendError(bot.Client(), err)
			defer os.Exit(1)
		}
		ltf.Println("closer mainCancel()")
		mainCancel()
		ltf.Println("closer ips.close")
		ips.close()
		wg.Wait()
		// pressEnter()
	})
	ul, err = jibber_jabber.DetectLanguage()
	if err != nil {
		ul = "en"
	}
	if len(chats) == 0 {
		err = Errorf(dic.add(ul,
			"en:Usage: %s AllowedChatID1 AllowedChatID2 AllowedChatIDx\n",
			"ru:Использование: %s РазрешённыйChatID1 РазрешённыйChatID2 РазрешённыйChatIDх\n",
		), os.Args[0])
		return
	} else {
		li.Println(dic.add(ul,
			"en:Allowed ChatID:",
			"ru:Разрешённые ChatID:",
		), chats)
	}
	ex, err := os.Getwd()
	if err == nil {
		mxbPingJson = filepath.Join(ex, mxbPingJson)
	}
	li.Println(filepath.FromSlash(mxbPingJson))

	bot, err = CreateBotWithProxy(os.Getenv("TOKEN"))

	if err != nil {
		err = srcError(err)
		return
	}

	botInfo, err := bot.Client().GetBot(mainCtx)
	if err != nil {
		err = srcError(err)
		return
	}
	me = botInfo

	// Настраиваем обработчик ошибок
	bot.OnError = func(err error, c maxigobot.Context) {
		let.Println("Bot error:", err)
		SendError(c.API(), err)
	}

	// Регистрируем обработчики
	setupHandlers()

	tacker = time.NewTicker(tt)
	defer tacker.Stop()

	wg.Add(1)
	go saver()

	wg.Add(1)
	// main loop
	go func() {
		defer wg.Done()
		ticker = time.NewTicker(dd)
		defer ticker.Stop()
		defer tacker.Stop()
		for {
			select {
			case <-mainCtx.Done():
				ltf.Println("Ticker done")
				return
			case t := <-ticker.C:
				ltf.Println("Tick at", t)
				ips.update(customer{})
			case t := <-tacker.C:
				ltf.Println("Tack at", t)
				ttCancel()
				ttCtx, ttCancel = context.WithCancel(mainCtx)
				wg.Add(1)
				go func() {
					defer wg.Done()
					// Бот maxigo-bot сам обрабатывает рестарт через long polling
					// Нам просто нужно перезапустить ticker
					restart(tacker, tt)
				}()
			}
		}
	}()

	err = loader()
	if err != nil {
		return
	}

	// Запускаем бота (блокирует)
	wg.Add(1)
	go func() {
		defer wg.Done()
		bot.Start()
	}()

	closer.Hold()
}

// setupHandlers настраивает все обработчики бота
func setupHandlers() {
	// Обработка текстовых сообщений (не команд)
	bot.Handle(maxigobot.OnText, func(c maxigobot.Context) error {
		text := c.Text()

		// Проверяем на IP адреса
		if reIP.MatchString(text) {
			return handleIPMessage(c)
		}

		// Проверяем на дату (пасхалка) только в личных чатах
		if reYYYYMMDD.MatchString(text) && isPrivateChatFromContext(c) {
			return handleEasterEgg(c)
		}

		// Проверяем на ответ "-" для удаления
		if c.Message().Link != nil && c.Message().Link.Type == "reply" {
			replyMsgID := c.Message().Link.Message.MID
			_, err := c.API().DeleteMessage(mainCtx, replyMsgID)
			if err != nil {
				let.Println(err)
				// Если не удалось удалить, редактируем текст
				body := &maxigo.NewMessageBody{Text: maxigo.Some("-")}
				_, err = c.API().EditMessage(mainCtx, replyMsgID, body)
				if err != nil {
					let.Println(err)
				}
			}
		}

		return nil
	})

	// Обработка callback от кнопок
	bot.Handle(maxigobot.OnCallback(""), func(c maxigobot.Context) error {
		return handleCallback(c)
	})

	// Обработка добавления пользователя
	bot.Handle(maxigobot.OnUserAdded, func(c maxigobot.Context) error {
		if !chats.allowed(c.Chat()) {
			return nil
		}

		text := fmt.Sprintf("%s\n%s\n%s",
			dic.add(ul, "en:Hello villagers!", "ru:Здорово, селяне!\n"),
			dic.add(ul, "en:Is carriage ready?\n", "ru:Карета готова?\n"),
			dic.add(ul, "en:The cart is ready!🏓", "ru:Телега готова!🏓"),
		)

		return c.Send(text)
	})

	// Обработка удаления пользователя
	bot.Handle(maxigobot.OnUserRemoved, func(c maxigobot.Context) error {
		if !chats.allowed(c.Chat()) {
			return nil
		}

		text := fmt.Sprintf("%s\n%s\n%s",
			dic.add(ul, "en:He flew away, but promised to return❗\n    ", "ru:Он улетел, но обещал вернуться❗\n    "),
			dic.add(ul, "en:Cute...", "ru:Милый..."),
			dic.add(ul, "en:Cute...", "ru:Милый..."),
		)

		return c.Send(text)
	})

	// Обработка запуска бота (кнопка Start)
	bot.Handle(maxigobot.OnBotStarted, func(c maxigobot.Context) error {
		payload := c.Payload()
		if payload == "" {
			return nil
		}

		ds, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil
		}

		ltf.Println(string(ds))
		text := "/start " + string(ds)

		// Создаем временный контекст для обработки
		if reYYYYMMDD.MatchString(text) {
			// Создаем фейковый message для пасхалки
			tempMsg := c.Message()
			if tempMsg != nil {
				tempText := text
				tempMsg.Body.Text = &tempText
				return handleEasterEgg(c)
			}
		} else if reIP.MatchString(text) {
			// Создаем фейковый message для IP
			tempMsg := c.Message()
			if tempMsg != nil {
				tempText := text
				tempMsg.Body.Text = &tempText
				return handleIPMessage(c)
			}
		}

		return nil
	})

	// Команда /start
	bot.Handle("/start", func(c maxigobot.Context) error {
		if !isPrivateChatFromContext(c) {
			return nil
		}

		payload := c.Payload()
		if payload != "" {
			// Обрабатываем payload как IP или дату
			if reYYYYMMDD.MatchString(payload) {
				tempMsg := c.Message()
				if tempMsg != nil {
					tempMsg.Body.Text = &payload
					return handleEasterEgg(c)
				}
			} else if reIP.MatchString(payload) {
				tempMsg := c.Message()
				if tempMsg != nil {
					tempMsg.Body.Text = &payload
					return handleIPMessage(c)
				}
			}
		}

		return handleHelp(c)
	})

	// Команда /restart (только для владельца)
	bot.Handle("/restart", func(c maxigobot.Context) error {
		if chats.allowed(c.Sender().UserID) {
			restart(tacker, tt)
		}
		return nil
	})

	// Команда /stop (только для владельца)
	bot.Handle("/stop", func(c maxigobot.Context) error {
		if chats.allowed(c.Sender().UserID) {
			closer.Close()
		}
		return nil
	})
}

// handleCallback обрабатывает нажатие на inline кнопку
func handleCallback(c maxigobot.Context) error {
	userID := c.Sender().UserID
	chatID := c.Chat()
	_, ups := allowed(ul, userID, chatID)

	data := c.Data()
	ip := ""
	if c.Message() != nil {
		ip = reIP.FindString(c.Text())
	}

	responseText := ups + tf(ips.count() == 0, "∅", ip+data)

	// Отвечаем на callback
	if err := c.Respond(responseText); err != nil {
		let.Println(err)
	}

	// Проверяем права
	my := chats.allowed(userID)
	if c.Message() != nil && !isPrivateChatFromContext(c) {
		my = true // В группах разрешаем всем
	}

	if !my {
		return nil
	}

	// Обрабатываем команды кнопок
	if data == "❎" {
		return c.Delete()
	}

	if data == "…" {
		// Кнопка "..." - показать больше кнопок
		attachments, _ := c.Message().Body.ParseAttachments()
		for _, att := range attachments {
			if kb, ok := att.(*maxigo.InlineKeyboardAttachment); ok {
				newButtons := kb.Payload.Buttons
				if len(newButtons) == 1 && ips.count() > 0 {
					// Показываем все кнопки
					newButtons = [][]maxigo.Button{append(newButtons[0], ikbs[:len(ikbs)-1]...)}
				}
				if err := c.Edit("", maxigobot.WithAttachments(maxigo.NewInlineKeyboardAttachment(newButtons))); err != nil {
					let.Println(err)
				}
				return nil
			}
		}
		return nil
	}

	if ips.count() == 0 {
		return nil
	}

	if strings.HasPrefix(data, "…") {
		ips.update(customer{Cmd: strings.TrimPrefix(data, "…")})
	} else {
		ips.write(ip, customer{Cmd: data})
	}

	return nil
}

// handleIPMessage обрабатывает сообщение с IP адресами
func handleIPMessage(c maxigobot.Context) error {
	userID := c.Sender().UserID
	chatID := c.Chat()
	text := c.Text()

	ok, ups := allowed(ul, userID, chatID)
	keys, _ := set(reIP.FindAllString(text, -1))
	ltf.Println("handleIPMessage", keys)

	if ok {
		for _, ip := range keys {
			ips.write(ip, customer{Tm: c.Message()})
		}
		return nil
	}

	ikbsf = len(ikbs) - 1
	news := ""
	for _, ip := range keys {
		if ips.read(ip) {
			ips.write(ip, customer{Tm: c.Message()})
		} else {
			news += ip + " "
		}
	}

	if len(news) > 1 {
		msgText := fmt.Sprintf("`/%s`\n%s", strings.TrimRight(news, " "), ups)
		if err := c.Send(msgText, maxigobot.WithAttachments(maxigo.NewInlineKeyboardAttachment([][]maxigo.Button{ikbs[ikbsf:]}))); err != nil {
			let.Println(err)
		}
	}

	return nil
}

// handleHelp отправляет справку
func handleHelp(c maxigobot.Context) error {
	if !isPrivateChatFromContext(c) {
		return nil
	}

	userID := c.Sender().UserID
	chatID := c.Chat()

	// Отправляем справку
	ok, ups := allowed(ul, userID, chatID)

	helpText := fmt.Sprintf("%s\n`/127.0.0.1 127.0.0.2 127.0.0.254`\n%s",
		dic.add(ul, "en:List of IP addresses expected\n", "ru:Ожидался список IP адресов\n"),
		ups,
	)

	var buttons [][]maxigo.Button
	if chats.allowed(userID) && ips.count() > 0 {
		buttons = [][]maxigo.Button{ikbs}
	} else if ok {
		buttons = [][]maxigo.Button{}
	} else {
		buttons = [][]maxigo.Button{ikbs}
	}

	if err := c.Send(helpText, maxigobot.WithAttachments(maxigo.NewInlineKeyboardAttachment(buttons))); err != nil {
		let.Println(err)
	}

	return nil
}

// handleEasterEgg обрабатывает пасхалку с датами
func handleEasterEgg(c maxigobot.Context) error {
	if !isPrivateChatFromContext(c) {
		return nil
	}

	text := c.Text()
	keys, _ := set(reYYYYMMDD.FindAllString(text, -1))
	ltf.Println("handleEasterEgg", keys)

	for _, key := range keys {
		fss := reYYYYMMDD.FindStringSubmatch(key)
		if len(fss) < 3 {
			continue
		}

		bd, err := time.ParseInLocation("20060102150405", strings.Join(fss[2:], "")+"120000", time.Local)
		if err != nil {
			continue
		}

		nbd := fmt.Sprintf("%s %s", fss[1], bd.Format("2006-01-02"))
		tl := start(me, nbd)

		// Формируем текст с годовщинами
		var lines []string
		lines = append(lines, fmt.Sprintf("⚡ [%s](%s)", tl, tl))
		lines = append(lines, fmt.Sprintf("`%s`", nbd))
		lines = append(lines, fmt.Sprintf("🔗\nhttps://t.me/share/url?url=%s", tl))

		for _, year := range la(bd) {
			lines = append(lines, year+"\n")
		}

		msgText := strings.Join(lines, "\n")
		if err := c.Send(msgText); err != nil {
			let.Println(err)
		}
	}

	return nil
}

// isPrivateChatFromContext проверяет, является ли чат личным диалогом из контекста
func isPrivateChatFromContext(c maxigobot.Context) bool {
	if c.Message() == nil {
		return false
	}
	return c.Message().Recipient.ChatType == maxigo.ChatDialog
}

// restart перезапускает ticker
func restart(t *time.Ticker, d time.Duration) {
	if t != nil {
		t.Reset(time.Millisecond * 100)
		time.Sleep(time.Millisecond * 150)
		t.Reset(d)
	}
}

// allowed проверяет разрешен ли ChatID
func allowed(key string, ChatIDs ...int64) (ok bool, s string) {
	s = "\n🏓"
	for _, v := range ChatIDs {
		ok = chats.allowed(v)
		if ok {
			return
		}
	}
	s = notAllowed(false, ChatIDs[0], key)
	return
}

// notAllowed возвращает сообщение о запрещённом доступе
func notAllowed(ok bool, ChatID int64, key string) (s string) {
	s = "\n🏓"
	if ok {
		return
	}
	s = dic.add(key,
		"en:\nNot allowed for you",
		"ru:\nБатюшка не благословляет Вас",
	)
	if ChatID != 0 {
		s += fmt.Sprintf(":%d", ChatID)
	}
	s += "\n🏓"
	return
}

// start кодирует данные для deep link
func start(me *maxigo.BotInfo, s string) string {
	username := "bot"
	if me.Username != nil {
		username = *me.Username
	}
	return fmt.Sprintf("https://t.me/%s?start=%s", username, base64.StdEncoding.EncodeToString([]byte(s)))
}
