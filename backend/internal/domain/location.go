package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxLocationNameLength = 100

var ErrInvalidLocation = errors.New("невалидное место")

type Location struct {
	Name      string
	Region    string
	Latitude  float64
	Longitude float64
	TimeZone  string
}

func (location Location) Validate() error {
	var problems []string

	if strings.TrimSpace(location.Name) == "" {
		problems = append(problems, "пустое название")
	}
	if utf8.RuneCountInString(location.Name) > MaxLocationNameLength || utf8.RuneCountInString(location.Region) > MaxLocationNameLength {
		problems = append(problems, fmt.Sprintf("название длиннее %d символов", MaxLocationNameLength))
	}
	if !(location.Latitude >= -90 && location.Latitude <= 90) || !(location.Longitude >= -180 && location.Longitude <= 180) {
		problems = append(problems, fmt.Sprintf("координаты вне диапазона: %v, %v", location.Latitude, location.Longitude))
	}
	// LoadLocation принимает "" и "Local" как пояс сервера, а нужен пояс города.
	if _, err := time.LoadLocation(location.TimeZone); err != nil || location.TimeZone == "" || location.TimeZone == "Local" {
		problems = append(problems, fmt.Sprintf("неизвестный часовой пояс %q", location.TimeZone))
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidLocation, strings.Join(problems, "; "))
	}
	return nil
}

// Zone считает место проверенным: у невалидного пояса вернёт UTC.
func (location Location) Zone() *time.Location {
	zone, err := time.LoadLocation(location.TimeZone)
	if err != nil {
		return time.UTC
	}
	return zone
}
