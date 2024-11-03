package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/gjolly/google-rotation-planner/cmd/google-rotation-planner/localcred"
	calendar "google.golang.org/api/calendar/v3"
	yaml "gopkg.in/yaml.v3"
)

const week = 7 * 24 * time.Hour

func createShift(c *Config, member Member, frequence int, startDate time.Time, insertEvent func(*calendar.Event) error) error {
	endDate := startDate.Add(time.Duration(c.ShiftDuration * int(week)))
	tz, err := time.LoadLocation("")
	if err != nil {
		return err
	}
	fmt.Printf("creating shift for %v, first shift starting on %v and finshing on %v (%v week(s))\n", member.Name, startDate, endDate, c.ShiftDuration)

	event := &calendar.Event{
		Summary:     fmt.Sprintf("%v on duty", member.Name),
		Description: c.Description,
		Start: &calendar.EventDateTime{
			Date:     startDate.Format("2006-01-02"),
			TimeZone: tz.String(),
		},
		End: &calendar.EventDateTime{
			Date:     endDate.Format("2006-01-02"),
			TimeZone: tz.String(),
		},
		Recurrence: []string{fmt.Sprintf("RRULE:FREQ=WEEKLY;INTERVAL=%v", frequence)},
		Attendees: []*calendar.EventAttendee{
			{
				Email:       member.EmailAddr,
				DisplayName: member.Name,
			},
		},
		Transparency: "transparent",
	}

	if len(c.Attachments) != 0 {
		event.Attachments = make([]*calendar.EventAttachment, len(c.Attachments))
		for i, attachment := range c.Attachments {
			event.Attachments[i] = &calendar.EventAttachment{
				FileUrl: attachment.URL,
				Title:   attachment.Name,
			}
		}
	}

	err = insertEvent(event)
	if err != nil {
		return fmt.Errorf("unable to create event: %w", err)
	}

	return nil
}

func createRota(c *Config, insertEvent func(*calendar.Event) error) error {
	for shiftNum, member := range c.Members {
		startDate := c.StartDate.Add(time.Duration(shiftNum * c.ShiftDuration * int(week)))
		err := createShift(c, member, len(c.Members), startDate, insertEvent)
		if err != nil {
			return fmt.Errorf("failed to create shift for %v: %w", member, err)
		}
	}

	return nil
}

// Member is a persone part of the rota
type Member struct {
	Name      string `yaml:"name"`
	EmailAddr string `yaml:"email"`
}

func (m Member) String() string {
	return fmt.Sprintf("%v (%v)", m.Name, m.EmailAddr)
}

type Attachment struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Config holds the informations about the google calendar
// to use, the members of the rota and other paramaters
// for the rota
type Config struct {
	StartDate     time.Time    `yaml:"startDate"`
	Members       []Member     `yaml:"members"`
	CalendarID    string       `yaml:"calendarID"`
	ShiftDuration int          `yaml:"shiftDuration"`
	Description   string       `yaml:"description"`
	Notify        bool         `yaml:"notify"`
	Attachments   []Attachment `yaml:"attachments"`
}

func parseConfig(configPath string) (*Config, error) {
	c := new(Config)

	content, err := os.ReadFile(configPath) // the file is inside the local directory
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(content, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func cleanup(listEvents func() (*calendar.Events, error), deleteEvent func(*calendar.Event) error) error {
	events, err := listEvents()
	if err != nil {
		return fmt.Errorf("unable to retrieve next ten of the user's events: %w", err)
	}

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			fmt.Println("deleting ", item.Summary)
			err = deleteEvent(item)
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	return nil
}

type ConfigProvider interface {
	InitConfig(string) error
	Service(context.Context, string) (*calendar.Service, error)
}

var configProvider ConfigProvider = new(localcred.Provider)

var FlagCleanup = flag.Bool("cleanup", false, "Remove all the recurring events from the calendar.")

var generateListEvents = func(srv *calendar.EventsService, c *Config) func() (*calendar.Events, error) {
	return func() (*calendar.Events, error) {
		t := time.Now().Format(time.RFC3339)
		return srv.List(c.CalendarID).ShowDeleted(false).TimeMin(t).Do()
	}
}

var generateDeleteEvent = func(srv *calendar.EventsService, c *Config) func(event *calendar.Event) error {
	return func(event *calendar.Event) error {
		return srv.Delete(c.CalendarID, event.Id).Do()
	}
}

var generateCreateEvents = func(srv *calendar.EventsService, c *Config) func(event *calendar.Event) error {
	return func(event *calendar.Event) error {
		_, err := srv.Insert(c.CalendarID, event).
			SupportsAttachments(true).
			SendNotifications(c.Notify).
			Do()

		return err
	}
}

var getEventService = func(ctx context.Context) (*calendar.EventsService, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find user's home folder: %w", err)
	}
	cfgDir := path.Join(home, ".google-rotation-planner")

	err = configProvider.InitConfig(cfgDir)
	if err != nil {
		return nil, fmt.Errorf("unable to init config: %w", err)
	}

	srv, err := configProvider.Service(ctx, cfgDir)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}

	return srv.Events, nil
}

var configPath = "config.yaml"

func main() {
	flag.Parse()

	ctx := context.Background()

	srv, err := getEventService(ctx)
	if err != nil {
		log.Fatal(err)
	}

	c, err := parseConfig(configPath)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}

	if *FlagCleanup {
		err = cleanup(generateListEvents(srv, c), generateDeleteEvent(srv, c))
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	err = createRota(c, generateCreateEvents(srv, c))
	if err != nil {
		log.Fatalf("Failed to create rota: %v", err)
	}
}
