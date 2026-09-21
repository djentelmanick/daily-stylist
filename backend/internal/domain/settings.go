package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var DefaultSendAt = DayTime{Hour: 7}

var ErrInvalidSettings = errors.New("невалидные настройки")

// Без даты и пояса: «семь утра» у каждого пользователя наступают в свой момент.
type DayTime struct {
	Hour   int
	Minute int
}

type Settings struct {
	MorningEnabled bool
	SendAt         DayTime
}

func DefaultSettings() Settings {
	return Settings{MorningEnabled: true, SendAt: DefaultSendAt}
}

func (settings Settings) Validate() error {
	return settings.SendAt.Validate()
}

func (dayTime DayTime) Validate() error {
	if dayTime.Hour < 0 || dayTime.Hour > 23 || dayTime.Minute < 0 || dayTime.Minute > 59 {
		return fmt.Errorf("%w: время %d:%d вне суток", ErrInvalidSettings, dayTime.Hour, dayTime.Minute)
	}
	return nil
}

func (dayTime DayTime) String() string {
	return fmt.Sprintf("%02d:%02d", dayTime.Hour, dayTime.Minute)
}

func ParseDayTime(value string) (DayTime, error) {
	hourText, minuteText, found := strings.Cut(strings.TrimSpace(value), ":")
	hour, hourErr := strconv.Atoi(hourText)
	minute, minuteErr := strconv.Atoi(minuteText)
	if !found || hourErr != nil || minuteErr != nil {
		return DayTime{}, fmt.Errorf("%w: время %q не в формате ЧЧ:ММ", ErrInvalidSettings, value)
	}

	dayTime := DayTime{Hour: hour, Minute: minute}
	if err := dayTime.Validate(); err != nil {
		return DayTime{}, err
	}
	return dayTime, nil
}
