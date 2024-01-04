package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/gjolly/google-rotation-planner/cmd/google-rotation-planner/localcred"
	"github.com/pkg/errors"
	"google.golang.org/api/calendar/v3"
	"gopkg.in/yaml.v3"
)

const week = 7 * 24 * time.Hour

func createShift(c *Config, member Member, frequence int, startDate time.Time, srv *calendar.Service) error {
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

	_, err = srv.Events.Insert(c.CalendarID, event).SendNotifications(c.Notify).Do()
	if err != nil {
		return errors.Wrap(err, "unable to create event")
	}
	return nil
}

func createRota(c *Config, srv *calendar.Service) error {
	for shiftNum, member := range c.Members {
		startDate := c.StartDate.Add(time.Duration(shiftNum * c.ShiftDuration * int(week)))
		err := createShift(c, member, len(c.Members), startDate, srv)
		if err != nil {
			return errors.Wrapf(err, "failed to create shift for %v", member)
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

// Config holds the informations about the google calendar
// to use, the members of the rota and other paramaters
// for the rota
type Config struct {
	StartDate     time.Time `yaml:"startDate"`
	Members       []Member  `yaml:"members"`
	CalendarID    string    `yaml:"calendarID"`
	ShiftDuration int       `yaml:"shiftDuration"`
	Description   string    `yaml:"description"`
	Notify        bool      `yaml:"notify"`
}

func parseConfig() (*Config, error) {
	c := new(Config)

	content, err := ioutil.ReadFile("config.yaml") // the file is inside the local directory
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(content, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func cleanup(c *Config, srv *calendar.Service) error {
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(c.CalendarID).ShowDeleted(false).TimeMin(t).Do()

	if err != nil {
		return errors.Wrap(err, "Unable to retrieve next ten of the user's events")
	}

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		for _, item := range events.Items {
			fmt.Println("deleting ", item.Summary)
			err = srv.Events.Delete(c.CalendarID, item.Id).Do()
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	return nil
}

var FlagCleanup = flag.Bool("cleanup", false, "Remove all the recurring events from the calendar.")

func main() {
	flag.Parse()

	ctx := context.Background()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Cannot find user's home folder: %v", err)
	}
	cfgDir := path.Join(home, ".google-rotation-planner")

	provider := new(localcred.Provider)
	err = provider.InitConfig(cfgDir)
	if err != nil {
		log.Fatalf("Unable to init config: %v", err)
	}

	srv, err := provider.Service(ctx, cfgDir)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	c, err := parseConfig()
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}

	if *FlagCleanup {
		err = cleanup(c, srv)
		if err != nil {
			log.Fatal(err)
		}

		return
	}
	err = createRota(c, srv)
	if err != nil {
		log.Fatalf("Failed to create rota: %v", err)
	}
}
