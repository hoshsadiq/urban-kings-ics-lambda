package main

import (
	"log"
	"time"
	"strings"

	"net/url"
	"net/http"

	"github.com/satori/go.uuid"
	"github.com/jinzhu/now"
	"github.com/jordic/goics"
	"github.com/PuerkitoBio/goquery"
)

var URL = "http://urbankingsgym.com/timetable/"

type Event struct {
	DayOfWeek      int       `json:"day_of_week"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	ClassName      string    `json:"class_name"`
	ClassUrl       *url.URL  `json:"class_url"`
	InstructorName string    `json:"instructor_name"`
	InstructorUrl  *url.URL  `json:"instructor_url"`
	PaygOpen       bool      `json:"payg_open"`
}

type Events []Event

func main() {
	http.HandleFunc("/events.ics", getIcsEvents)
	http.HandleFunc("/", http.NotFound)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func getIcsEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Only GET", http.StatusMethodNotAllowed)

		return
	}

	w.Header().Set("Content-type", "text/calendar")
	w.Header().Set("charset", "utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("filename", "events.ics")

	events := getEvents()

	goics.NewICalEncode(w).Encode(events)
}

func getEvents() Events {
	response, err := http.Get(URL)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatalf("status code error: %d %s", response.StatusCode, response.Status)
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	idMapping := map[string]int{
		"tablepress-1": 0,
		"tablepress-2": 1,
		"tablepress-3": 2,
		"tablepress-4": 3,
		"tablepress-5": 4,
		"tablepress-6": 5,
		"tablepress-7": 6,
	}

	events := Events{}
	doc.Find(".tablepress tbody tr").Each(func(i int, el *goquery.Selection) {
		id, _ := el.Closest("table").First().Attr("id")
		dayOfWeek := idMapping[id]

		class := el.Find("td.column-3")
		classUrl := anchorHrefAsUrl(class)

		instructor := el.Find("td.column-4")
		instructorUrl := anchorHrefAsUrl(instructor)

		start := getTimeObject(strings.TrimSpace(el.Find("td.column-1").Text()), dayOfWeek)
		end := getTimeObject(strings.TrimSpace(el.Find("td.column-2").Text()), dayOfWeek)
		events = append(events, Event{
			DayOfWeek:      dayOfWeek,
			Start:          start,
			End:            end,
			ClassName:      strings.TrimSpace(class.Text()),
			ClassUrl:       classUrl,
			InstructorName: strings.TrimSpace(instructor.Text()),
			InstructorUrl:  instructorUrl,
			PaygOpen:       strings.ToLower(strings.TrimSpace(el.Find("td.column-5").Text())) == "yes",
		})
	})

	return events
}

func getTimeObject(timeStr string, dayOfWeek int) time.Time {
	//now.WeekStartDay = time.Monday
	date := now.Monday().AddDate(0, 0, dayOfWeek)
	timeParsed := now.MustParse(timeStr)

	return time.Date(date.Year(), date.Month(), date.Day(), timeParsed.Hour() - 1, timeParsed.Minute(), timeParsed.Second(), 0, date.Location())
}

func anchorHrefAsUrl(sel *goquery.Selection) *url.URL {
	a := sel.Find("a")
	if a.Length() != 1 {
		return nil
	}

	href, exists := a.Attr("href")
	if !exists {
		return nil
	}

	urlParsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return nil
	}

	return urlParsed
}

func (ev *Event) EmitICal() goics.Componenter {
	event := goics.NewComponent()

	uid, err := uuid.NewV4()
	uuid.Must(uid, err)

	event.SetType("VEVENT")
	k, v := goics.FormatDateTime("DTSTART", ev.Start)
	event.AddProperty(k, v)
	k, v = goics.FormatDateTime("DTEND", ev.End)
	event.AddProperty(k, v)
	event.AddProperty("UID", uid.String())
	event.AddProperty("DESCRIPTION", "some description")
	event.AddProperty("SUMMARY", ev.ClassName)
	event.AddProperty("LOCATION", "Urban Kings London")

	return event
}

func (rc Events) EmitICal() goics.Componenter {

	c := goics.NewComponent()
	c.SetType("VCALENDAR")
	c.AddProperty("CALSCAL", "GREGORIAN")
	c.AddProperty("PRODID", "-//hosh.io//Golang//EN")
	c.AddProperty("X-WR-CALNAME", "Urban Kings Classes")
	c.AddProperty("X-WR-TIMEZONE", "Europe/London")

	for _, ev := range rc {
		c.AddComponent(ev.EmitICal())
	}

	return c

}
