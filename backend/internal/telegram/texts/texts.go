package texts

import "fmt"

// Название кнопки меню задаётся в BotFather
const MenuButton = "Гардероб"

const (
	MoreOutfits = "Ещё варианты"
	Wear        = "Надеваю"
	Worn        = "Записал. В ближайшие дни постараюсь не повторять эту одежду."
	WearFailed  = "Не получилось записать. Попробуйте в приложении."
)

const OpenApp = "Собрать гардероб"

func Welcome(firstName string) string {
	greeting := "Привет!"
	if firstName != "" {
		greeting = fmt.Sprintf("Привет, %s!", firstName)
	}

	return fmt.Sprintf(`%s

Я Daily Stylist - каждое утро подбираю образ из вашего гардероба по погоде.

Начнём с гардероба: добавьте вещи, которые носите чаще всего. Проще всего сфотографировать - форма заполнится сама. Потом приложение всегда под кнопкой «%s» слева от поля ввода.`, greeting, MenuButton)
}
