package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Category string

const (
	CategoryNationalHoliday  Category = "national_holiday"
	CategoryReligiousHoliday Category = "religious_holiday"
	CategoryJointHoliday     Category = "joint_holiday"
	CategoryInternal         Category = "internal"
	CategoryBigEvent         Category = "big_event"
	CategoryMidterm          Category = "midterm"
	CategoryFinal            Category = "final"
	CategorySeminar          Category = "seminar"
)

func (c Category) Valid() bool {
	switch c {
	case CategoryNationalHoliday, CategoryReligiousHoliday, CategoryJointHoliday,
		CategoryInternal, CategoryBigEvent, CategoryMidterm, CategoryFinal, CategorySeminar:
		return true
	}
	return false
}

// DefaultColor returns the brand-token hex for a given category. Admin
// overrides via the per-event `color` field are honored on write; this
// is only used when the client did not supply one.
func (c Category) DefaultColor() string {
	switch c {
	case CategoryNationalHoliday:
		return "#f94141"
	case CategoryReligiousHoliday:
		return "#0f8657"
	case CategoryJointHoliday:
		return "#f7bf33"
	case CategoryInternal:
		return "#3a6dc5"
	case CategoryBigEvent:
		return "#0e1116"
	case CategoryMidterm:
		return "#fee5e5"
	case CategoryFinal:
		return "#c01f1f"
	case CategorySeminar:
		return "#ecf1fa"
	}
	return "#3a6dc5"
}

type LocationKind string

const (
	LocationPhysical LocationKind = "physical"
	LocationOnline   LocationKind = "online"
	LocationHybrid   LocationKind = "hybrid"
)

func (l LocationKind) Valid() bool {
	return l == LocationPhysical || l == LocationOnline || l == LocationHybrid
}

type Attachment struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"`
}

type Event struct {
	ID                uuid.UUID       `json:"id"`
	ParentEventID     *uuid.UUID      `json:"parent_event_id,omitempty"`
	Title             string          `json:"title"`
	Category          Category        `json:"category"`
	Color             string          `json:"color"`
	DescriptionJSON   json.RawMessage `json:"description_json,omitempty"`
	ThumbnailURL      *string         `json:"thumbnail_url,omitempty"`
	StartDatetime     time.Time       `json:"start_datetime"`
	EndDatetime       time.Time       `json:"end_datetime"`
	IsAllDay          bool            `json:"is_all_day"`
	Location          *string         `json:"location,omitempty"`
	LocationType      LocationKind    `json:"location_type"`
	MeetingLink       *string         `json:"meeting_link,omitempty"`
	Dresscode         *string         `json:"dresscode,omitempty"`
	Attendees         []string        `json:"attendees,omitempty"`
	Attachments       []Attachment    `json:"attachments"`
	RecurrenceRule    *string         `json:"recurrence_rule,omitempty"`
	RecurrenceEndDate *time.Time      `json:"recurrence_end_date,omitempty"`
	IsSeeded          bool            `json:"is_seeded"`
	IsPublished       bool            `json:"is_published"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
