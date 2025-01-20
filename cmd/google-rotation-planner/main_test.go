package main

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/net/context"
	calendar "google.golang.org/api/calendar/v3"
)

func generateNoopListEvents(*calendar.EventsService, *Config) func() (*calendar.Events, error) {
	return func() (*calendar.Events, error) {
		return nil, nil
	}
}

func generateNoopDeleteEvent(*calendar.EventsService, *Config) func(event *calendar.Event) error {
	return func(*calendar.Event) error {
		return nil
	}
}

func getNilEventsService(context.Context) (*calendar.EventsService, error) {
	return nil, nil
}

func TestEnd2End(t *testing.T) {
	generateListEvents = generateNoopListEvents
	generateDeleteEvent = generateNoopDeleteEvent
	getEventService = getNilEventsService

	config := `
shiftDuration: 1
startDate: 2024-11-03
calendarID: calendarID
attachments:
  - name: attachmentName
    url: fileURL
members:
  - name: Test User 1
    email: test1@example.com
  - name: Test User 2
    email: test2@example.com
`

	eventsCreated := 0
	generateCreateEvents = func(*calendar.EventsService, *Config) func(*calendar.Event) error {
		return func(event *calendar.Event) error {
			eventsCreated++
			if !strings.Contains(event.Summary, "Test User 1") && !strings.Contains(event.Summary, "Test User 2") {
				t.Error("event summary does not contain participant name")
			}

			if len(event.Attachments) != 1 {
				t.Errorf("%d attachments were expected but got %d", 1, len(event.Attachments))
				attachment := event.Attachments[0]

				if attachment.FileUrl != "fileURL" {
					t.Errorf("'%s' url was expected but got '%s'", "fileURL", attachment.FileUrl)
				}

				if attachment.Title != "attachmentName" {
					t.Errorf("'%s' title was expected for attachment but got '%s'", "attachmentName", attachment.Title)
				}
			}
			return nil
		}
	}

	configFile, err := os.CreateTemp("", "test-google-rotation-planner-*.yaml")
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// modify module variable
	configPath = configFile.Name()
	defer os.Remove(configPath)

	_, err = configFile.Write([]byte(config))
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err = configFile.Close()
	if err != nil {
		t.Fatalf("failed to close test file: %v", err)
	}

	main()

	if eventsCreated != 2 {
		t.Errorf("%d events should be created but got %d events", 2, eventsCreated)
	}
}

func TestCustomTitle(t *testing.T) {
	generateListEvents = generateNoopListEvents
	generateDeleteEvent = generateNoopDeleteEvent
	getEventService = getNilEventsService

	config := `
shiftDuration: 1
startDate: 2024-11-03
calendarID: calendarID
attachments:
  - name: attachmentName
    url: fileURL
members:
  - name: Test User 1
    email: test1@example.com
title: foobar {{.Name}}
`

	eventsCreated := 0
	generateCreateEvents = func(*calendar.EventsService, *Config) func(*calendar.Event) error {
		return func(event *calendar.Event) error {
			eventsCreated++
			expected := "foobar Test User 1"
			if event.Summary != expected {
				t.Errorf("event summary doesn't match the title, expected '%s' got '%s'", expected, event.Summary)
			}

			return nil
		}
	}

	configFile, err := os.CreateTemp("", "test-google-rotation-planner-*.yaml")
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// modify module variable
	configPath = configFile.Name()
	defer os.Remove(configPath)

	_, err = configFile.Write([]byte(config))
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	err = configFile.Close()
	if err != nil {
		t.Fatalf("failed to close test file: %v", err)
	}

	main()

	if eventsCreated != 1 {
		t.Errorf("%d events should be created but got %d events", 1, eventsCreated)
	}
}
