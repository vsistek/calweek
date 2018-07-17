package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"
)

type event struct {
	summary  string
	start    time.Time
	end      time.Time
    relevant bool
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// get start and end time of the current week
func weekBoundaries() (time.Time, time.Time) {
	start := time.Now()
	var startunix, endunix int64

	for start.Weekday() != time.Sunday {
		startunix = start.Unix()
		startunix -= 1
		start = time.Unix(startunix, 0)
	}
	startunix = start.Unix()
	startunix += 1
	start = time.Unix(startunix, 0)

	endunix = startunix + (7 * 24 * 60 * 60) - 1
	end := time.Unix(endunix, 0)

	return start, end
}

func decrease1Day(date time.Time) time.Time {
	dateunix := date.Unix()
	dateunix -= 24 * 60 * 60
	return time.Unix(dateunix, 0)
}

// return [] of events relevant to this week filtered by a given text in summary
func thisWeekEventsWithText(data string, text string) []event {
	var response []event
	var startdate, enddate string
	var item event

	startweek, endweek := weekBoundaries()

	// iterate through iCal data and select relevant events
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
        switch line {
        case "BEGIN:VEVENT":
            // new event, reset relevancy
            item.relevant = false
        case "END:VEVENT":
            if item.relevant {
                response = append(response, item)
            }
        default:
            switch strings.Split(line, ":")[0] {
            case "DTSTART;VALUE=DATE":
                // event start date, check whether it is relevant for this week
                startdate = strings.Split(line, ":")[1]
                item.start, _ = time.Parse("20060102", startdate)
                if item.start.Unix() < endweek.Unix() {
                    item.relevant = true
                }
            case "DTEND;VALUE=DATE":
                // event end date, check whether it is relevant for this week
                enddate = strings.Split(line, ":")[1]
                item.end, _ = time.Parse("20060102", enddate)
                // decrease end date by 1 day, as the iCal events end on date when
                // the event is not relevant anymore.
                item.end = decrease1Day(item.end)
                if item.end.Unix() >= startweek.Unix() && item.relevant {
                    item.relevant = true
                } else {
                    item.relevant = false
                }

            case "SUMMARY":
                item.summary = strings.TrimPrefix(line, "SUMMARY:")
                if item.relevant && !strings.Contains(item.summary, text) {
                    item.relevant = false
                }
            }
        }
	}
	return response
}

func printEvents(events []event) {
	var eventstrings []string
	var eventstr string

	for _, event := range events {
		eventstr = event.start.Format("01-02-2006")
		if event.start.Format("01-02-2006") == event.end.Format("01-02-2006") {
			eventstr += ":              "
		} else {
			eventstr += " - "
			eventstr += event.end.Format("01-02-2006")
			eventstr += ": "
		}
		eventstr += event.summary
		eventstrings = append(eventstrings, eventstr)
	}
	sort.Strings(eventstrings)
	for _, item := range eventstrings {
		fmt.Println(item)
	}
}

func main() {
	// read simple configuration file that contains only secret CalDav URL
	usr, err := user.Current()
	check(err)
	url, err := ioutil.ReadFile(usr.HomeDir + "/.calweek")
	check(err)
	urlstr := string(url[:])
	urlstr = strings.TrimSuffix(urlstr, "\n")

	// first argument is a text filter for event summary
	var text string
	if len(os.Args) == 1 {
		text = ""
	} else {
		text = os.Args[1]
	}
	// using http, download everything from the caldav server
	resp, err := http.Get(urlstr)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	check(err)

	// process the response body and get relevant events
	myevents := thisWeekEventsWithText(string(body[:]), text)

	printEvents(myevents)
}
