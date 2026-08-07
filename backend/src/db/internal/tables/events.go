package tables

import (
	"fmt"
)

func (q *Tables) InitializeEventsTable() error {
	var err error
	// Events table:
	// id parent_id start_timestamp end_timestamp calendar settings
	//
	// The parent is the "original" event in case of recurrences.
	// The timestamps are included for future-proofing.
	_, err = q.Tx.Exec(
		q.Context,
		`
		CREATE TABLE events (
			id UUID PRIMARY KEY,
			parent_id UUID REFERENCES events(id) ON DELETE CASCADE,
			start_timestamp TIMESTAMP NOT NULL,
			end_timestamp TIMESTAMP NOT NULL,
			calendar UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
			settings JSONB NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("could not create events table: %v", err)
	}

	_, err = q.Tx.Exec(
		q.Context,
		`
		CREATE INDEX index_events_calendar ON events (calendar);
	`)
	if err != nil {
		return fmt.Errorf("could not create secondary index on events table: %v", err)
	}

	return nil
}

func (q *Tables) InitializeEventOverridesTable() error {
	var err error
	// Calendars table:
	// id title description color
	_, err = q.Tx.Exec(
		q.Context,
		`
		CREATE TABLE event_overrides (
			eventid UUID NOT NULL UNIQUE REFERENCES events(id) ON DELETE CASCADE,
			title TEXT,
			description TEXT,
			color BYTEA,
			future BOOLEAN
		);
	`)
	if err != nil {
		return fmt.Errorf("could not create event overrides table: %v", err)
	}

	return nil
}
