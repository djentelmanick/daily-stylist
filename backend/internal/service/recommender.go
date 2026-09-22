package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/djentelmanick/daily-stylist/backend/internal/domain"
	"github.com/djentelmanick/daily-stylist/backend/internal/domain/rules"
)

var ErrWeatherUnavailable = errors.New("сервис погоды недоступен")

type Recommendation struct {
	Location domain.Location
	Weather  domain.Weather
	Outfits  []domain.Outfit
	Notes    []domain.Note
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

func (recommender *Recommender) Recommend(ctx context.Context, userID int64) (Recommendation, error) {
	situation, err := recommender.situation(ctx, userID)
	if err != nil {
		return Recommendation{}, fmt.Errorf("подбор образа: %w", err)
	}
	outfits, notes := rules.Recommend(situation.input)
	return Recommendation{
		Location: situation.location,
		Weather:  situation.input.Weather,
		Outfits:  outfits,
		Notes:    notes,
	}, nil
}

// Candidates - чем заменить вещь replaceID в образе, а при replaceID = 0 - что в него добавить.
func (recommender *Recommender) Candidates(
	ctx context.Context,
	userID int64,
	outfitIDs []int64,
	replaceID int64,
) ([]rules.Candidate, error) {
	situation, err := recommender.situation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("варианты замены: %w", err)
	}

	outfit := chosen(situation.input.Items, outfitIDs)
	if replaceID == 0 {
		return rules.Additions(situation.input, outfit), nil
	}
	index := slices.IndexFunc(outfit, func(item domain.Item) bool { return item.ID == replaceID })
	if index < 0 {
		return nil, fmt.Errorf("варианты замены: %w", ErrItemNotFound)
	}
	return rules.Replacements(situation.input, outfit, outfit[index]), nil
}

func (recommender *Recommender) Review(ctx context.Context, userID int64, itemIDs []int64) ([]domain.Note, error) {
	situation, err := recommender.situation(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("разбор образа: %w", err)
	}
	return rules.Review(situation.input.Weather, chosen(situation.input.Items, itemIDs)), nil
}

type situation struct {
	location domain.Location
	input    rules.Input
}

func (recommender *Recommender) situation(ctx context.Context, userID int64) (situation, error) {
	location, err := recommender.locations.Get(ctx, userID)
	if err != nil {
		return situation{}, err
	}
	now := recommender.now().In(location.Zone())

	from, to := domain.ForecastWindow(now)
	weather, err := recommender.forecaster.Forecast(ctx, location, from, to)
	if err != nil {
		return situation{}, fmt.Errorf("%w: %w", ErrWeatherUnavailable, err)
	}

	items, err := recommender.items.ListByUser(ctx, userID)
	if err != nil {
		return situation{}, err
	}

	today := domain.DateOf(now)
	lastWorn, err := recommender.outfits.LastWorn(ctx, userID, today.AddDate(0, 0, -rules.HistoryDays), today)
	if err != nil {
		return situation{}, err
	}
	wornDaysAgo := make(map[int64]int, len(lastWorn))
	for itemID, day := range lastWorn {
		wornDaysAgo[itemID] = int(today.Sub(day).Hours() / 24)
	}

	return situation{
		location: location,
		input: rules.Input{
			Items:       items,
			Weather:     weather,
			Season:      domain.SeasonAt(now),
			WornDaysAgo: wornDaysAgo,
		},
	}, nil
}

func chosen(items []domain.Item, itemIDs []int64) []domain.Item {
	var result []domain.Item
	for _, item := range items {
		if slices.Contains(itemIDs, item.ID) {
			result = append(result, item)
		}
	}
	return result
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
