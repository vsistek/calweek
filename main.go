package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"sort"
	"strings"
	"time"
)

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

// return []strings of events relevant to this week filtered by a given text in summary
func thisWeekEventsWithText(data string, text string) []string {
	var response []string
	var relevant bool
	var event, startdatestr, enddatestr, summarystr string
	var startdate, enddate time.Time

	// get week start and end time for later comparisons
	startweek, endweek := weekBoundaries()

	// prepare regexps for matching
	var begin = regexp.MustCompile(`^BEGIN\:VEVENT$`)
	var dtstart = regexp.MustCompile(`^DTSTART;VALUE=DATE:.+$`)
	var dtend = regexp.MustCompile(`^DTEND;VALUE=DATE:.+$`)
	var summary = regexp.MustCompile(`^SUMMARY:.+$`)
	var end = regexp.MustCompile(`^END\:VEVENT$`)

	// iterate through iCal data and select relevant events
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case begin.MatchString(line):
			// new event, reset relevancy
			relevant = false

		case dtstart.MatchString(line):
			// event start date, check whether it is relevant for this week
			startdatestr = strings.Split(line, ":")[1]
			startdate, _ = time.Parse("20060102", startdatestr)
			if startdate.Unix() < endweek.Unix() {
				relevant = true
			}

		case dtend.MatchString(line):
			// event end date, check whether it is relevant for this week
			enddatestr = strings.Split(line, ":")[1]
			enddate, _ = time.Parse("20060102", enddatestr)
			// decrease end date by 1 day, as the iCal events end on date when
			// the event is not relevant anymore.
			enddate = decrease1Day(enddate)
			if enddate.Unix() >= startweek.Unix() && relevant {
				relevant = true
			} else {
				relevant = false
			}

		case summary.MatchString(line):
			summarystr = strings.TrimPrefix(line, "SUMMARY:")
			if relevant && !strings.Contains(summarystr, text) {
				relevant = false
			}

		case end.MatchString(line):
			if relevant {
				event = startdate.Format("01-02-2006")
                if startdate.Format("01-02-2006") == enddate.Format("01-02-2006") {
                    event += ":              "
                } else {
                    event += " - "
                    event += enddate.Format("01-02-2006")
                    event += ": "
                }
				event += summarystr
				response = append(response, event)
			}
		}
	}
	return response
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

	// process the body and get relevant and nicely formatted events
	myevents := thisWeekEventsWithText(string(body[:]), text)
	// sort events alphadetically
	sort.Strings(myevents)
	for _, event := range myevents {
		fmt.Println(event)
	}
}
