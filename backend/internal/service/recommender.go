package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain/rules"
)

var ErrWeatherUnavailable = errors.New("сервис погоды недоступен")

type Recommendation struct {
	Location domain.Location
	Weather  domain.Weather
	Outfits  []domain.Outfit
	// Почему образов нет.
	Notes []domain.Note
}

type Recommender struct {
	items      ItemRepository
	outfits    OutfitRepository
	locations  LocationRepository
	forecaster Forecaster
	now        func() time.Time
}

func NewRecommender(
	items ItemRepository,
	outfits OutfitRepository,
	locations LocationRepository,
	forecaster Forecaster,
	now func() time.Time,
) *Recommender {
	return &Recommender{items: items, outfits: outfits, locations: locations, forecaster: forecaster, now: now}
}

func (recommender *Recommender) Recommend(ctx context.Context, userID int64) (recommendation Recommendation, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("подбор образа: %w", err)
		}
	}()

	location, err := recommender.locations.Get(ctx, userID)
	if err != nil {
		return Recommendation{}, err
	}
	now := recommender.now().In(location.Zone())

	from, to := domain.ForecastWindow(now)
	weather, err := recommender.forecaster.Forecast(ctx, location, from, to)
	if err != nil {
		return Recommendation{}, fmt.Errorf("%w: %w", ErrWeatherUnavailable, err)
	}

	items, err := recommender.items.ListByUser(ctx, userID)
	if err != nil {
		return Recommendation{}, err
	}

	// Сегодняшний образ в историю не входит: открыв подбор после «Надеваю»,
	// человек увидит тот же образ, а не штраф за него.
	today := domain.DateOf(now)
	lastWorn, err := recommender.outfits.LastWorn(ctx, userID, today.AddDate(0, 0, -rules.HistoryDays), today)
	if err != nil {
		return Recommendation{}, err
	}
	wornDaysAgo := make(map[int64]int, len(lastWorn))
	for itemID, day := range lastWorn {
		wornDaysAgo[itemID] = int(today.Sub(day).Hours() / 24)
	}

	outfits, notes := rules.Recommend(rules.Input{
		Items:       items,
		Weather:     weather,
		Season:      domain.SeasonAt(now),
		WornDaysAgo: wornDaysAgo,
	})
	return Recommendation{Location: location, Weather: weather, Outfits: outfits, Notes: notes}, nil
}

func (recommender *Recommender) TodayOutfit(ctx context.Context, userID int64) ([]domain.Item, error) {
	location, err := recommender.locations.Get(ctx, userID)
	// Без города образ не записать, значит, и показывать нечего.
	if errors.Is(err, ErrLocationNotSet) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("образ дня: %w", err)
	}

	items, err := recommender.outfits.WornOn(ctx, userID, domain.DateOf(recommender.now().In(location.Zone())))
	if err != nil {
		return nil, fmt.Errorf("образ дня: %w", err)
	}
	domain.SortForWearing(items)
	return items, nil
}

func (recommender *Recommender) WearToday(ctx context.Context, userID int64, itemIDs []int64) error {
	if len(itemIDs) == 0 {
		return nil
	}
	location, err := recommender.locations.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("запись образа: %w", err)
	}
	today := domain.DateOf(recommender.now().In(location.Zone()))
	if err := recommender.outfits.SaveWorn(ctx, userID, today, itemIDs); err != nil {
		return fmt.Errorf("запись образа: %w", err)
	}
	return nil
}
