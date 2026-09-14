package texts

import "fmt"

// Название кнопки меню задаётся в BotFather
const MenuButton = "Гардероб"

func Welcome(firstName string) string {
	greeting := "Привет!"
	if firstName != "" {
		greeting = fmt.Sprintf("Привет, %s!", firstName)
	}

	return fmt.Sprintf(`%s

Я Daily Stylist - подбираю образ на день из вашего гардероба с учётом погоды.

Сейчас можно собрать гардероб: добавить вещи и отметить грязные. Подбор образа скоро появится!

Нажмите «%s» слева от поля ввода.`, greeting, MenuButton)
}
