package app

import (
	"context"
	"errors"
	"strings"

	"goask/internal/data"
	"goask/internal/service"
)

func (a *application) openConversation(id int64) {
	msgs, err := a.db.Messages(id)
	if err != nil {
		a.note.Set(err.Error())
		return
	}
	if a.conversation != id {
		a.stopChat()
		a.question.Set("")
	}
	a.conversation = id
	a.messages = msgs
	a.messagesVersion.Update(increment)
}
func (a *application) newConversation() {
	a.stopChat()
	a.conversation = 0
	a.messages = nil
	a.question.Set("")
	a.partial.Set("")
	a.messagesVersion.Update(increment)
	a.switchMode("chat")
}
func (a *application) stopChat() {
	if a.cancelChat == nil {
		return
	}
	a.cancelChat()
	a.finishChat(nil)
}
func (a *application) finishChat(err error) {
	a.generationID++
	var notices []string
	if text := a.partial.Get(); text != "" {
		id, saveErr := a.db.AddMessage(a.conversation, "assistant", text)
		if saveErr != nil {
			notices = append(notices, "回答未保存："+saveErr.Error())
		}
		a.messages = append(a.messages, data.Message{ID: id, Role: "assistant", Content: text})
	}
	a.cancelChat = nil
	a.generating.Set(false)
	a.partial.Set("")
	if err != nil && !errors.Is(err, context.Canceled) {
		notices = append(notices, "问答失败："+err.Error())
	}
	if len(notices) > 0 {
		a.chatErrors[a.conversation] = strings.Join(notices, "\n")
	}
	a.messagesVersion.Update(increment)
}
func (a *application) send() {
	text := strings.TrimSpace(a.question.Get())
	if text == "" || a.generating.Get() {
		return
	}
	a.loadCredentials(a.cx)
	providerID := a.chatProviderID()
	provider, _ := service.ProviderByID(providerID)
	cfg, configErr := a.providerConfig(providerID)
	if !provider.AI || configErr != nil {
		a.note.Set("请先在设置中配置并选择问答提供商。")
		return
	}
	if a.conversation == 0 {
		id, err := a.db.NewConversation(text)
		if err != nil {
			a.note.Set(err.Error())
			return
		}
		a.conversation = id
	}
	id, err := a.db.AddMessage(a.conversation, "user", text)
	if err != nil {
		a.note.Set(err.Error())
		return
	}
	a.messages = append(a.messages, data.Message{ID: id, Role: "user", Content: text})
	history := make([]service.Message, 0, len(a.messages))
	for _, m := range a.messages {
		history = append(history, service.Message{Role: m.Role, Content: m.Content})
	}
	a.question.Set("")
	a.note.Set("")
	delete(a.chatErrors, a.conversation)
	a.partial.Set("")
	a.generating.Set(true)
	a.messagesVersion.Update(increment)
	ctx, cancel := context.WithCancel(a.lifetime)
	a.cancelChat = cancel
	a.generationID++
	run := a.generationID
	go func() {
		defer cancel()
		err := service.Chat(ctx, cfg, history, func(delta string) {
			a.cx.Post(func() {
				if run == a.generationID {
					a.partial.Set(a.partial.Get() + delta)
				}
			})
		})
		a.cx.Post(func() {
			if run == a.generationID {
				a.finishChat(err)
			}
		})
	}()
}
