package helpers

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/dustin/go-humanize"
)

var funcMap = template.FuncMap{
	"add":          add,
	"minus":        minus,
	"divide":       divide,
	"multiply":     multiply,
	"capitalize":   capitalize,
	"formatDate":   formatDate,
	"cleanText":    formatText,
	"isLive":       isTimePassed,
	"greater":      greater,
	"formatLink":   FormatLink,
	"addComma":     addComma,
	"poolDuration": poolDuration,
	"convertTime":  convertTime,
	"timeTo":       timeTo,
	"calcPercent":  calcPercent,
	"isValid":      IsEmptyOrUndefined,
}

func add(a, b int) int { return a + b }

func minus(a, b float64) float64 { return a - b }

func divide(a, b float64) string {
	d := a / b
	formatted := fmt.Sprintf("%.2f", d)

	parts := strings.Split(formatted, ".")

	val, _ := strconv.ParseInt(parts[0], 10, 64)
	return strings.Replace(humanize.Comma(val)+"."+parts[1], "-", "", 1)
}

func multiply(a, b float64) string {
	d := a * b
	formatted := fmt.Sprintf("%.2f", d)
	parts := strings.Split(formatted, ".")

	val, _ := strconv.ParseInt(parts[0], 10, 64)
	return strings.Replace(humanize.Comma(val)+"."+parts[1], "-", "", 1)
}
func addComma(val interface{}) string {
	switch val.(type) {
	case int:
		return humanize.Comma(int64(val.(int)))
	case float64:
		return humanize.Commaf(val.(float64))
	default:
		return humanize.Comma(val.(int64))

	}
}

func greater(a, b float64) bool {
	if a > b {
		return true
	} else {
		return false
	}
}

func capitalize(s string) string {
	if len(s) < 1 {
		return s
	}
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}

func formatDate(t string) string {
	layout := "2006-01-02T15:04:05Z"
	d, err := time.Parse(layout, t)
	if err != nil {
		log.Printf("Error parsing date: %v", err)
		return t
	}
	var val = d.UTC().Format("Monday, 02 Jan - 15:04")

	parsedYear, parsedMonth, parsedDay := d.Date()

	currentYear, currentMonth, currentDay := time.Now().Date()
	if parsedYear == currentYear && parsedMonth == currentMonth && parsedDay == currentDay {
		val = "Today " + d.UTC().Format("15:04")
	}
	return val
}

func formatText(s string) string {
	words := strings.Fields(s)
	filteredWords := []string{}
	for _, word := range words {
		if !strings.HasPrefix(word, "#") {
			cleanedWord := strings.ReplaceAll(word, "@", "")
			filteredWords = append(filteredWords, cleanedWord)
		}
	}
	return strings.Join(filteredWords, " ")
}

func isTimePassed(timeStr string) (bool, error) {
	layout := "2006-01-02T15:04:05Z"

	parsedTime, err := time.Parse(layout, timeStr)
	if err != nil {
		return false, fmt.Errorf("error parsing time: %v", err)
	}

	currentTime := time.Now().UTC()

	return parsedTime.Before(currentTime), nil
}

func convertTime(seconds int64) string {
	var timeRemaining string
	days := seconds / (24 * 60 * 60)
	if days > 1 {
		timeRemaining = timeRemaining + fmt.Sprintf("%d", days) + "days "
	}

	hours := (seconds % (24 * 60 * 60)) / (60 * 60)
	if hours > 1 {
		timeRemaining = timeRemaining + fmt.Sprintf("%d", hours) + "hrs "
	}

	minutes := (seconds % (60 * 60)) / 60
	if minutes > 1 {
		timeRemaining = timeRemaining + fmt.Sprintf("%d", minutes) + "min "
	}

	sec := (seconds % (60)) / 60
	if sec > 60 {
		timeRemaining = timeRemaining + fmt.Sprintf("%d", seconds) + "sec "
	}
	return timeRemaining
}

func poolDuration(pool map[string]interface{}) string {
	marketplace := strings.ToLower(pool["marketplace"].(string))
	if marketplace == "citrus" {
		return "7 - 14 days"
	} else if marketplace == "banx" && pool["collectionName"].(string) == "Flip loans (Pool)" {
		return "7/14 days"
	} else if marketplace == "banx" {
		return "Perpetual"
	} else {
		return convertTime(int64(pool["duration"].(float64)))
	}
}

func timeTo(endDateStr string) string {
	seconds := TimeDiff(endDateStr)

	return convertTime(seconds)
}

func calcPercent(val float64, percent float64) string {
	equiv := val - ((val * percent) / 100)

	return addComma(math.Round(equiv))
}

func FormatLink(s string) string { return strings.ReplaceAll(s, " ", "-") }

func IsEmptyOrUndefined(value interface{}) bool {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.String && v.Len() == 0 {
		return false
	}
	return true
}

func FormatHTML(data []map[string]interface{}, tmplFileName string) (string, error) {

	tmplPath := filepath.Join("tmpl", tmplFileName)

	t, err := template.New(tmplFileName).Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		return "", err
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		log.Printf("Error executing template: %v", err)
		return "", err
	}

	return tpl.String(), nil
}
