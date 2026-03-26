package main

import (
	"fmt"
	"strings"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// send ip to ch for add it to ping list
func worker(ip string, ch cCustomer) {
	var (
		err error
		status,
		statusOld string
		tl       = start(me, ip)
		deadline = time.Now().Add(dd)
		cus      = customers{}
		ikbs     = []maxigo.Button{
			maxigo.NewCallbackButton("🔁", "🔁"),
			maxigo.NewCallbackButton("🔂", "🔂"),
			maxigo.NewCallbackButton("⏸️", "⏸️"),
			maxigo.NewCallbackButton("❌", "❌"),
			maxigo.NewCallbackButton("…", "…"),
			maxigo.NewCallbackButton("❎", "❎"),
		}
		ikbsf int
	)
	defer ips.del(ip, false)
	for {
		select {
		case <-mainCtx.Done():
			for i, cu := range cus {
				tempMsg := &maxigo.Message{
					Body: maxigo.MessageBody{MID: cu.Tm.Body.MID},
					Recipient: maxigo.Recipient{
						ChatID:   cu.Tm.Recipient.ChatID,
						ChatType: cu.Tm.Recipient.ChatType,
					},
				}
				if cu.Tm.Sender != nil {
					tempMsg.Sender = &maxigo.User{UserID: cu.Tm.Sender.UserID}
				}
				cu.Tm = tempMsg
				cu.Cmd = ip
				if i == 0 {
					if cu.Tm.Sender != nil {
						cu.Tm.Sender.FirstName = status
					}
					ltf.Println("saved ", ip, status, deadline)
				}
				save <- cu
			}
			ltf.Println("done", ip)
			return
		case cust, ok := <-ch:
			if !ok {
				ltf.Println("channel closed", ip)
				return
			}
			if cust.Tm == nil { // update
				switch cust.Cmd {
				case "⏸️":
					deadline = time.Now().Add(-refresh)
				case "🔁":
					deadline = time.Now().Add(dd)
				case "🔂":
					deadline = time.Now().Add(refresh)
				default:
					if strings.HasSuffix(cust.Cmd, "❌") {
						tsX := strings.TrimSuffix(cust.Cmd, "❌")
						if tsX == "" || strings.HasSuffix(status, tsX) || strings.HasPrefix(status, tsX) || (strings.HasPrefix(status, "❗") && tsX == "❗") {
							for _, cu := range cus {
								ltf.Println("bot.DeleteMessage", cu)
								re := cu.Reply
								if re != nil {
									bot.Client().DeleteMessage(mainCtx, re.Body.MID)
								}
							}
							return
						}
					}
				}
			} else { // load
				if cust.Cmd == ip && cust.Tm.Timestamp > 0 {
					if cust.Tm.Sender != nil {
						status = cust.Tm.Sender.FirstName
					}
					deadline = time.Unix(cust.Tm.Timestamp, 0)
					ltf.Println("loaded ", ip, status, deadline)
				}
				cus = append(cus, cust)
			}
			statusOld = status
			ltf.Println(ip, cust, len(ch), status, time.Now().Before(deadline))
			if time.Now().Before(deadline) {
				status, err = ping(ip)
				if err != nil {
					status = "❗"
					ltf.Println("ping", ip, err)
				}
			} else {
				if !strings.HasSuffix(status, "⏸️") {
					status += "⏸️"
				}
			}
			for i, cu := range cus {
				re := cu.Reply
				chatID := int64(0)
				senderID := int64(0)
				privateChat := false

				if cu.Tm.Sender != nil {
					senderID = cu.Tm.Sender.UserID
				}
				if cu.Tm.Recipient.ChatID != nil {
					chatID = *cu.Tm.Recipient.ChatID
				}
				privateChat = cu.Tm.Recipient.ChatType == maxigo.ChatDialog

				ltf.Println(i, chatID, senderID, ip, status, statusOld)
				if re == nil || status != statusOld {
					if re != nil {
						bot.Client().DeleteMessage(mainCtx, re.Body.MID)
					}
					ikbsf = 0
					if !chats.allowed(tf(privateChat, senderID, chatID)) {
						ikbsf = len(ikbs) - 1
					}
					msgText := fmt.Sprintf("%s\n`%s`\n[⚡](%s)", status, ip, tl)

					var buttons [][]maxigo.Button
					if ikbsf == 0 {
						buttons = [][]maxigo.Button{ikbs}
					} else {
						buttons = [][]maxigo.Button{ikbs[ikbsf:]}
					}

					cus[i].Reply, err = bot.Client().SendMessage(mainCtx, chatID, &maxigo.NewMessageBody{
						Text: maxigo.Some(msgText),
						Attachments: []maxigo.AttachmentRequest{
							maxigo.NewInlineKeyboardAttachment(buttons),
						},
						Link: &maxigo.NewMessageLink{
							Type: "reply",
							MID:  getMessageID(cu.Tm),
						},
					})
					if err != nil {
						letf.Println("delete", ip)
						ips.del(ip, false)
					}
				}
			}
		}
	}
}
