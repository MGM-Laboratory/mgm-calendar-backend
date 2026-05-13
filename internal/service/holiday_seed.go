package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"mgm.lab/calendar-backend/internal/model"
	"mgm.lab/calendar-backend/internal/repository"
)

type HolidaySeeder struct {
	pool   *pgxpool.Pool
	repo   *repository.EventRepository
	apiURL string
}

func NewHolidaySeeder(pool *pgxpool.Pool, repo *repository.EventRepository, apiURL string) *HolidaySeeder {
	return &HolidaySeeder{pool: pool, repo: repo, apiURL: apiURL}
}

type apiHoliday struct {
	HolidayDate       string `json:"holiday_date"`
	HolidayName       string `json:"holiday_name"`
	IsNationalHoliday bool   `json:"is_national_holiday"`
}

func (s *HolidaySeeder) SeedIfNeeded(ctx context.Context) error {
	now := time.Now()
	for _, year := range []int{now.Year(), now.Year() + 1} {
		name := fmt.Sprintf("holidays_%d", year)
		has, err := s.repo.HasSeed(ctx, name)
		if err != nil {
			return fmt.Errorf("check seed %s: %w", name, err)
		}
		if has {
			continue
		}
		count, err := s.seedYear(ctx, year)
		if err != nil {
			return fmt.Errorf("seed %d: %w", year, err)
		}
		if err := s.repo.RecordSeed(ctx, name); err != nil {
			return fmt.Errorf("record seed %s: %w", name, err)
		}
		log.Printf("seeded %d Indonesian holidays for %d", count, year)
	}
	return nil
}

func (s *HolidaySeeder) seedYear(ctx context.Context, year int) (int, error) {
	url := fmt.Sprintf("%s?year=%d", s.apiURL, year)
	client := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return 0, fmt.Errorf("api status %d: %s", resp.StatusCode, string(body))
	}

	var holidays []apiHoliday
	if err := json.NewDecoder(resp.Body).Decode(&holidays); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.UTC
	}

	count := 0
	for _, h := range holidays {
		t, ok := parseFlexibleDate(h.HolidayDate, loc)
		if !ok {
			continue
		}
		start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		end := start.Add(24*time.Hour - time.Second)
		cat := classifyHoliday(h.HolidayName, h.IsNationalHoliday)

		e := &model.Event{
			Title:         strings.TrimSpace(h.HolidayName),
			Category:      cat,
			Color:         cat.DefaultColor(),
			StartDatetime: start,
			EndDatetime:   end,
			IsAllDay:      true,
			LocationType:  model.LocationPhysical,
			IsSeeded:      true,
			IsPublished:   true,
			Attachments:   []model.Attachment{},
		}
		if err := s.repo.Create(ctx, e); err != nil {
			return count, fmt.Errorf("insert %q: %w", h.HolidayName, err)
		}
		count++
	}
	return count, nil
}

func parseFlexibleDate(s string, loc *time.Location) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02", "2006-1-2", "2006-1-02", "2006-01-2"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

var religiousKeywords = []string{
	"idul fitri", "idul adha", "maulid", "isra", "miraj", "mi'raj",
	"natal", "nyepi", "waisak", "kenaikan yesus", "wafat yesus",
	"wafat isa", "kenaikan isa", "imlek",
	"tahun baru hijriah", "tahun baru islam", "tahun baru saka",
	"paskah",
}

func classifyHoliday(name string, isNational bool) model.Category {
	if !isNational {
		return model.CategoryJointHoliday
	}
	lower := strings.ToLower(name)
	for _, k := range religiousKeywords {
		if strings.Contains(lower, k) {
			return model.CategoryReligiousHoliday
		}
	}
	return model.CategoryNationalHoliday
}
