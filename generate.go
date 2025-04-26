package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Lofanmi/chinese-calendar-golang/calendar"
	"github.com/signintech/gopdf"
)

const (
	layout     = "20060102 1504"
	font       = "fontDefault"
	center     = "center"
	cellH      = 13.0
	startMainX = 30.0
	startLeftX = 445.0
)

var (
	solarColar         = gopdf.RGBColor{R: 0, G: 0, B: 128}
	lunarColar         = gopdf.RGBColor{R: 180, G: 130, B: 255}
	defaultBorderStyle = gopdf.BorderStyle{
		Bottom:   true,
		Width:    0.5,
		RGBColor: blackColor,
	}
	blackColor = gopdf.RGBColor{R: 0, G: 0, B: 0}
	chars      = map[int][]string{
		1: {"a", "i", "j", "q", "y"},
		2: {"b", "k", "r"},
		3: {"g", "l"},
		4: {"d", "m"},
		5: {"h", "n"},
		6: {"s", "u", "v", "w"},
		7: {"o", "x", "e", "z"},
		8: {"c", "p", "f"},
		9: {"t"},
	}
	charMap = revert(chars)
)

func generatePdf(name string, nickName string, birthday string) ([]byte, bool, int) {
	// birthday = "2024/09/29 10:18"
	solarT, err := time.Parse(layout, birthday)
	if err != nil {
		slog.Warn("Error parsing time:", "birthday", birthday, "err", err)
		return nil, false, http.StatusBadRequest
	}
	lunarT, leapSolar := getLunarTime(solarT)
	pdf := NewPdf()

	generate(pdf, name, nickName, solarT, lunarT)
	isLeap := leapSolar != nil
	if isLeap {
		leapLunar := time.Date(lunarT.Year(), lunarT.Month()+1, lunarT.Day(), lunarT.Hour(), lunarT.Minute(), 0, 0, time.UTC)
		generate(pdf, name, nickName, *leapSolar, leapLunar)
	}

	var buf bytes.Buffer
	_, err = pdf.WriteTo(&buf)
	if err != nil {
		slog.Warn("Error writing PDF to buffer:", "err", err)
		return nil, false, http.StatusInternalServerError
	}
	return buf.Bytes(), isLeap, http.StatusOK
}

type TimeData struct {
	SumData
	Time      time.Time
	DigitMap  map[string]int
	MoreThan2 bool
}
type SumData struct {
	Year  int
	Month int
	Day   int
}

func generate(pdf *gopdf.GoPdf, name string, nickName string, solarT time.Time, lunarT time.Time) {
	pdf.AddPage()
	solar := newTimeData(solarT)
	lunar := newTimeData(lunarT)

	drawDegreeTables(pdf, solar, lunar)

	drawYearlyTables(pdf, solarT.Year(),
		[]int{solar.Month, solar.Day},
		[]int{lunar.Month, lunar.Day},
	)

	drawNameTable(pdf, name, nickName)

	drawMonthDayTable(pdf, 200,
		gopdf.RGBColor{R: 255, G: 246, B: 143},
		gopdf.RGBColor{R: 139, G: 40, B: 19},
		gopdf.RGBColor{R: 139, G: 117, B: 0},
		"流月", 12,
		[]int{solar.Year, solar.Day},
		[]int{lunar.Year, lunar.Day},
	)

	drawMonthDayTable(pdf, 380,
		gopdf.RGBColor{R: 193, G: 255, B: 193},
		gopdf.RGBColor{R: 0, G: 50, B: 0},
		gopdf.RGBColor{R: 34, G: 139, B: 34},
		"流日", 31,
		[]int{solar.Year, solar.Month},
		[]int{lunar.Year, lunar.Month},
	)
	// err = pdf.WritePdf("table.pdf")
	// if err != nil {
	// 	slog.Warn("Error writing PDF:", "err", err)
	// 	return http.StatusInternalServerError
	// }
}

var numRegex = regexp.MustCompile(`^\d+$`)

func generateNum(number string) ([]byte, int) {
	if !numRegex.MatchString(number) {
		slog.Warn("Invalid number:", "number", number)
		return nil, http.StatusBadRequest
	}

	sumNum, _ := sumDigitsByStr(number)
	result := sumDigitsToString(sumNum)
	type Result struct {
		LifePassword string `json:"lifePassword"`
	}
	buf, err := json.Marshal(&Result{
		LifePassword: result,
	})
	if err != nil {
		slog.Warn("Error marshalling result:", "err", err)
		return nil, http.StatusInternalServerError
	}
	return buf, http.StatusOK
}

func NewPdf() *gopdf.GoPdf {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	err := pdf.AddTTFFont(font, "./static/Arial Unicode.ttf")
	if err != nil {
		slog.Warn("Error adding font:", "err", err)
	}
	return pdf
}

func drawDegreeTables(pdf *gopdf.GoPdf, solar *TimeData, lunar *TimeData) {
	sumHr, _ := sumDigits(solar.Time.Hour())
	sumMin, _ := sumDigits(solar.Time.Minute())
	solars := [5]int{solar.Year, solar.Month, solar.Day, sumHr, sumMin}
	lunars := [5]int{lunar.Year, lunar.Month, lunar.Day, sumHr, sumMin}

	startY := 40.0
	drawDegreeHeader(pdf, startY, solar.Time, lunar.Time)
	startY += cellH
	titles := []string{"老年", "中年", "青年(主命)", "青少年", "幼年"}
	drawMainRow(pdf, startMainX, startY, 8.5, blackColor, titles)
	drawMainRow(pdf, 232.0, startY, 8.5, blackColor, titles)

	startY += cellH
	solarStages := stageNumbers("+", solars)
	lunarStages := stageNumbers("-", lunars)
	drawMainRow(pdf, startMainX, startY, 8.5, solarColar, solarStages)
	drawMainRow(pdf, 232.0, startY, 8.5, lunarColar, lunarStages)

	startY += cellH
	drawMainRow(pdf, startMainX, startY, 8.5, solarColar, soulDegrees(solar, solarStages))
	drawMainRow(pdf, 232.0, startY, 8.5, lunarColar, soulDegrees(lunar, lunarStages))
}

func drawDegreeHeader(pdf *gopdf.GoPdf, startY float64, solar time.Time, lunar time.Time) error {
	table := pdf.NewTableLayout(startMainX, startY, 12, 0)

	dayLayout := "2006/01/02"
	minuteLayout := "15:04"
	table.AddColumn("陽曆時間：", 65, center)
	table.AddColumn(solar.Format(dayLayout), 65, center)
	table.AddColumn(solar.Format(minuteLayout), 70, center)
	table.AddColumn("陰曆時間：", 65, "right")
	table.AddColumn(lunar.Format(dayLayout), 65, center)
	table.AddColumn(lunar.Format(minuteLayout), 78, "left")

	table.SetTableStyle(gopdf.CellStyle{
		BorderStyle: gopdf.BorderStyle{},
	})

	table.SetHeaderStyle(gopdf.CellStyle{
		BorderStyle: defaultBorderStyle,
		FillColor:   gopdf.RGBColor{R: 187, G: 255, B: 255},
		TextColor:   blackColor,
		Font:        font,
		FontSize:    9,
	})

	err := table.DrawTable()
	if err != nil {
		slog.Warn("Error drawing table:", "err", err)
		return err
	}
	return nil
}

func drawYearlyTables(pdf *gopdf.GoPdf, solarYear int,
	sumSolars []int, sumLunars []int,
) {
	startY := 120.0

	titles := []string{"歲次"}
	yrs := []string{"西元"}
	solars := []string{"主秘術"}
	lunars := []string{"陰曆"}
	for i := 1; i <= 100; i++ {
		if len(titles) == 10 {
			drawMainRow(pdf, startMainX, startY, 9, blackColor, titles)
			startY += cellH
			drawMainRow(pdf, startMainX, startY, 9, blackColor, yrs)
			startY += cellH
			drawMainRow(pdf, startMainX, startY, 9, solarColar, solars)
			startY += cellH
			drawMainRow(pdf, startMainX, startY, 9, lunarColar, lunars)

			titles = titles[:1]
			yrs = yrs[:1]
			solars = solars[:1]
			lunars = lunars[:1]
			startY += cellH + 10
		}
		yr := solarYear + i - 1
		titles = append(titles, fmt.Sprint(i))
		yrs = append(yrs, fmt.Sprint(yr))

		sumYr, _ := sumDigits(yr)
		solars = append(solars, finalNumber("+", append(sumSolars, sumYr)))
		lunars = append(lunars, finalNumber("-", append(sumLunars, sumYr)))
	}
}

func drawMainRow(pdf *gopdf.GoPdf, startX float64, startY float64, fontSize float64, textC gopdf.RGBColor, values []string) error {
	table := pdf.NewTableLayout(startX, startY, 12, 0)

	for i, v := range values {
		if len(values) == 5 && i == 2 {
			table.AddColumn(v, 44, center)
		} else {
			if i == 0 {
				table.AddColumn(v, 38, center)
			} else {
				table.AddColumn(v, 41, center)
			}
		}
	}

	table.SetTableStyle(gopdf.CellStyle{
		BorderStyle: gopdf.BorderStyle{},
	})

	table.SetHeaderStyle(gopdf.CellStyle{
		BorderStyle: defaultBorderStyle,
		FillColor:   gopdf.RGBColor{R: 255, G: 255, B: 255},
		TextColor:   textC,
		Font:        font,
		FontSize:    fontSize,
	})

	err := table.DrawTable()
	if err != nil {
		slog.Warn("Error drawing table:", "err", err)
		return err
	}
	return nil
}

func drawNameTable(pdf *gopdf.GoPdf, name string, nickName string) {
	startY, nameYGap, digitYGap := 45.0, 16.0, 4.0
	drawNameRow(pdf, 442, startY, 36, "英文名(護照)", 5.5, center)
	drawNameRow(pdf, 478, startY-nameYGap, 77, "", 9, "left", name)
	drawNameRow(pdf, 478, startY-digitYGap, 77, "", 7.2, "right", charsToDigits(name))
	startY += 23
	drawNameRow(pdf, 442, startY, 36, "英文名(小名)", 5.5, center)
	drawNameRow(pdf, 478, startY-nameYGap, 77, "", 9, "left", nickName)
	drawNameRow(pdf, 478, startY-digitYGap, 77, "", 7.2, "right", charsToDigits(nickName))
}

func drawNameRow(pdf *gopdf.GoPdf, startX float64, startY float64,
	width float64, name string, fontSize float64, align string, value ...string) error {
	table := pdf.NewTableLayout(startX, startY, 14, len(value))

	table.AddColumn(name, width, align)
	for _, v := range value {
		table.AddRow([]string{v})
	}

	table.SetTableStyle(gopdf.CellStyle{
		BorderStyle: gopdf.BorderStyle{},
	})
	table.SetHeaderStyle(gopdf.CellStyle{
		TextColor: blackColor,
		Font:      font,
		FontSize:  fontSize,
	})
	table.SetCellStyle(gopdf.CellStyle{
		FillColor: gopdf.RGBColor{R: 255, G: 255, B: 255},
		TextColor: blackColor,
		Font:      font,
		FontSize:  fontSize,
	})
	err := table.DrawTable()
	if err != nil {
		slog.Warn("Error drawing table:", "err", err)
		return err
	}
	return nil
}

func drawMonthDayTable(pdf *gopdf.GoPdf, startY float64,
	headerColor gopdf.RGBColor, solarColar gopdf.RGBColor, lunarColor gopdf.RGBColor,
	name string, total int,
	sumSolars []int, sumLunars []int) {
	titles := []string{}
	solars := []string{}
	lunars := []string{}
	for i := 1; i <= total; i++ {
		sumTarget, _ := sumDigits(i)
		titles = append(titles, fmt.Sprint(i))
		solars = append(solars, finalNumber("+", append(sumSolars, sumTarget)))
		lunars = append(lunars, finalNumber("-", append(sumLunars, sumTarget)))
	}
	if len(lunars) > 30 {
		lunars = lunars[:30]
	}

	startX, width := startLeftX, 29.0
	drawLeftTable(pdf, startX, startY, width, headerColor, blackColor, name, titles)
	startX += width
	width = 42
	drawLeftTable(pdf, startX, startY, width, headerColor, solarColar, "國曆", solars)
	startX += width
	drawLeftTable(pdf, startX, startY, width, headerColor, lunarColor, "農曆", lunars)
}

func drawLeftTable(pdf *gopdf.GoPdf,
	startX float64, startY float64, width float64,
	headerColor gopdf.RGBColor, textColor gopdf.RGBColor,
	name string, values []string) error {
	table := pdf.NewTableLayout(startX, startY, 12, len(values))

	table.AddColumn(name, width, center)
	for _, v := range values {
		table.AddRow([]string{v})
	}

	table.SetTableStyle(gopdf.CellStyle{
		BorderStyle: gopdf.BorderStyle{
			Top:      true,
			Left:     true,
			Right:    true,
			Width:    0.5,
			RGBColor: blackColor,
		},
	})

	table.SetHeaderStyle(gopdf.CellStyle{
		FillColor: headerColor,
		TextColor: blackColor,
		Font:      font,
		FontSize:  8.5,
	})

	table.SetCellStyle(gopdf.CellStyle{
		BorderStyle: gopdf.BorderStyle{
			Top:      true,
			Bottom:   true,
			Width:    0.5,
			RGBColor: blackColor,
		},
		FillColor: gopdf.RGBColor{R: 255, G: 255, B: 255},
		TextColor: textColor,
		Font:      font,
		FontSize:  8.5,
	})

	err := table.DrawTable()
	if err != nil {
		slog.Warn("Error drawing table:", "err", err)
		return err
	}
	return nil
}

func stageNumbers(symbol string, values [5]int) []string {
	all := [5]string{}
	for i := range values {
		all[i] = finalNumber(symbol, values[:i+1])
	}
	return all[:]
}

func finalNumber(symbol string, values []int) string {
	return symbol + sumDigitsToString(sumSlices(values))
}

func sumSlices(values []int) int {
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum
}

func intSliceToString(values []int) string {
	return strings.Trim(strings.Replace(fmt.Sprint(values), " ", "/", -1), "[]")
}

func sumDigitsToString(v int) string {
	ds := []int{v}
	for {
		d, _ := sumDigits(v)
		if len(ds) != 0 && ds[len(ds)-1] == d {
			break
		}
		ds = append(ds, d)
		v = d
	}
	return intSliceToString(ds)
}

func sumDigits(number int) (int, []int) {
	return sumDigitsByStr(fmt.Sprint(number))
}

func sumDigitsByStr(numberStr string) (int, []int) {
	sum := 0
	digits := []int{}
	for _, n := range numberStr {
		num, _ := strconv.Atoi(string(n))
		digits = append(digits, num)
		sum += num
	}
	return sum, digits
}

func getLunarTime(solar time.Time) (time.Time, *time.Time) {
	c := calendar.BySolar(int64(solar.Year()), int64(solar.Month()), int64(solar.Day()), int64(solar.Hour()), int64(solar.Minute()), 0)
	lunar := c.Lunar
	lunarTime := time.Date(int(lunar.GetYear()), time.Month(lunar.GetMonth()), int(lunar.GetDay()), int(solar.Hour()), solar.Minute(), 0, 0, time.UTC)
	if !lunar.IsLeapMonth() {
		return lunarTime, nil
	}
	newC := calendar.ByLunar(lunar.GetYear(), lunar.GetMonth()+1, lunar.GetDay(), int64(solar.Hour()), int64(solar.Minute()), 0, false)
	newSolar := time.Date(int(newC.Solar.GetYear()), time.Month(newC.Solar.GetMonth()), int(newC.Solar.GetDay()), int(solar.Hour()), solar.Minute(), 0, 0, time.UTC)
	return lunarTime, &newSolar
}

func newTimeData(v time.Time) *TimeData {
	sumYr, digitYr := sumDigits(v.Year())
	sumMon, digitMon := sumDigits(int(v.Month()))
	sumDay, digitDay := sumDigits(v.Day())
	digits := map[string]int{}

	allDigits := append(digitYr, append(digitMon, digitDay...)...)
	for _, v := range allDigits {
		digits[fmt.Sprint(v)]++
	}

	moreThan2 := false
	for _, v := range digits {
		if v > 2 {
			moreThan2 = true
			break
		}
	}

	return &TimeData{
		SumData: SumData{
			Year:  sumYr,
			Month: sumMon,
			Day:   sumDay,
		},
		Time:      v,
		DigitMap:  digits,
		MoreThan2: moreThan2,
	}
}

func revert(chars map[int][]string) map[string]int {
	charMap := map[string]int{}
	for k, vs := range chars {
		for _, v := range vs {
			charMap[v] = k
		}
	}
	return charMap
}

func charsToDigits(v string) string {
	vs := strings.Split(strings.ToLower(v), "")
	sum := 0
	for _, v := range vs {
		digit, ok := charMap[v]
		if ok {
			sum += digit
		}
	}
	return finalNumber("", []int{sum})
}

func soulDegrees(d *TimeData, values []string) []string {
	res := []string{}
	for _, v := range values {
		degree := soulDegree(v, d.DigitMap)
		if degree == 7 && d.MoreThan2 {
			degree = 6
		}
		res = append(res, fmt.Sprint(degree))
	}
	return res
}

func soulDegree(stage string, digits map[string]int) int {
	stage = strings.ReplaceAll(strings.ReplaceAll(stage, "+", ""), "-", "")
	layers := strings.Split(stage, "/")
	lens := len(layers)
	if lens == 1 {
		_, ok := digits[layers[0]]
		if ok {
			return 7
		}
		return 1
	}

	degree := 1
	for _, v := range strings.Split(layers[0], "") {
		_, ok := digits[v]
		if ok {
			degree += 1
		}
	}

	if lens == 2 {
		_, ok := digits[layers[1]]
		if ok {
			switch degree {
			case 1, 2:
				degree += 3
			case 3:
				degree = 7
			}
		}
		return degree
	}

	mainLayer := strings.Split(layers[1]+layers[2], "")
	for _, v := range mainLayer {
		_, ok := digits[v]
		if !ok {
			return degree
		}
	}
	switch degree {
	case 1, 2:
		degree += 3
	case 3:
		degree = 7
	}
	return degree
}
