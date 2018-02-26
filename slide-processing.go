package main

import (
	"fmt"
	"image"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//Find date of 72 hour slide in header by looking for month name
func findDateOfPhotoNodeSlides(slides []Slide) (slideDate time.Time, err error) {

	//Build months name array
	var monthsLong []string  //Long month ex:January
	var monthsShort []string //Short month ex:Jan
	for i := time.January; i <= time.December; i++ {
		monthsLong = append(monthsLong, i.String())
		monthsShort = append(monthsShort, i.String()[0:3])
	}

	//Store current and next month values
	var currentMonth time.Month
	var nextMonth time.Month
	currentMonth = time.Now().Month()
	if nextMonth = currentMonth + 1; nextMonth > time.December {
		nextMonth = time.January
	}

	//Search for the current and next month strings
	var closestMonthSpelling string
	var closestMonthSlide Slide
	monthsSearchArray := []string{monthsLong[currentMonth-1], monthsLong[nextMonth-1], monthsShort[currentMonth-1], monthsLong[nextMonth-1]}

	//Look through all slides (savetypes) to find closest date to time.Now. Assume any OCR error dates will be at higher date difference than a correct OCR date within ~72 hours.
	compareTargetDate := time.Now()
	for i, v := range monthsSearchArray {
		if closestMonthSpelling, closestMonthSlide, err = findKeywordClosestSpellingInPhotoInSaveImageTypes(v, slides); err != nil {
			return
		}

		//Determine which month we are trying to find in this loop
		var estimatedMonth time.Month
		if i%2 == 0 {
			estimatedMonth = currentMonth
		} else {
			estimatedMonth = nextMonth
		}

		if len(closestMonthSpelling) > 0 {
			//We found a close spelling, move onto finding bounding box

			//Find month bounds in hOCR
			var bbox image.Rectangle
			bbox, err = getTextBounds(closestMonthSlide.HOCRText, closestMonthSpelling)
			if err != nil {
				return
			}

			//Search for date in all slides using closest match spelling
			for _, s := range slides {
				var foundDate time.Time
				//Try to find date from uncropped image
				if foundDate, err = findDateFromPlainText(s.PlainText, closestMonthSpelling, estimatedMonth); err != nil {
					return
				}

				//Check if date is closest date found so far
				if slideDate.Equal(time.Time{}) || !foundDate.Equal(time.Time{}) {
					slideDate = foundDate
				} else {
					slideDate = closerDate(compareTargetDate, slideDate, foundDate)
				}

				if (slideDate.Equal(time.Time{}) || math.Abs(float64(compareTargetDate.Sub(foundDate))) < math.Abs(float64(compareTargetDate.Sub(slideDate)))) {
					slideDate = foundDate
				}

				//Try to find date on cropped image
				//Crop current slide to only show the image line that month was found on
				if err = runImageMagickDateCropProcess(s, bbox.Min.Y-5, (bbox.Max.Y-bbox.Min.Y)+10); err != nil {
					return
				}
				//Run OCR on cropped image for current slide using bbox
				copySlide := s
				copySlide.Suffix = IMAGE_SUFFIX_CROPPED
				if err = doOCRForSlide(&copySlide); err != nil {
					return
				}

				//Try to find date from cropped image
				if foundDate, err = findDateFromPlainText(copySlide.PlainText, closestMonthSpelling, estimatedMonth); err != nil {
					return
				}

				//Check if date is closest date found so far
				if slideDate.Equal(time.Time{}) || !foundDate.Equal(time.Time{}) {
					slideDate = foundDate
				} else {
					slideDate = closerDate(compareTargetDate, slideDate, foundDate)
				}
			}
		} else {
			//fuzzy match not found
		}
		//Try to find exact match with regex. Fuzzy match may fail or get bad results if month is between other letters.
		for _, s := range slides {
			var foundDate time.Time
			//Try to find date from uncropped image
			if foundDate, err = findDateFromPlainText(s.PlainText, v, estimatedMonth); err != nil {
				return
			}

			//Check if date is closest date found so far
			if slideDate.Equal(time.Time{}) || !foundDate.Equal(time.Time{}) {
				slideDate = foundDate
			} else {
				slideDate = closerDate(compareTargetDate, slideDate, foundDate)
			}

			if (slideDate.Equal(time.Time{}) || math.Abs(float64(compareTargetDate.Sub(foundDate))) < math.Abs(float64(compareTargetDate.Sub(slideDate)))) {
				slideDate = foundDate
			}
		}
	}

	//fmt.Printf("%v date %v bbox %v %v %v %v\n", closestMonthSlide.Terminal.Title, closestMonthSpelling, bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Max.Y)

	return
}

func findDateFromPlainText(plainText string, closestMonthSpelling string, estimatedMonth time.Month) (date time.Time, err error) {
	//Lowercase closestMonthSpelling
	closestMonthSpelling = strings.ToLower(closestMonthSpelling)

	//fmt.Println("find date with closestMonthSpelling ", closestMonthSpelling)
	//fmt.Println("plaintext", plainText)

	//Find date with Regexp
	var DMYRegex *regexp.Regexp
	var MDYRegex *regexp.Regexp
	//Match Date Month Year. Capture date and year
	if DMYRegex, err = regexp.Compile(fmt.Sprintf("([0-9]{1,2})[a-z]{0,3}%v([0-9]{2,4})", closestMonthSpelling)); err != nil {
		return
	}
	//Match Month Date Year. Capture date and year
	if MDYRegex, err = regexp.Compile(fmt.Sprintf("%v([0-9]{2})[a-z]{0,3}([0-9]{4})", closestMonthSpelling)); err != nil {
		return
	}

	//Remove common OCR errors from and lowercase input string
	r := strings.NewReplacer(
		".", "",
		",", "",
		" ", "")
	var input = strings.ToLower(r.Replace(plainText))
	var regexResult []string
	if regexResult = DMYRegex.FindStringSubmatch(input); len(regexResult) == 3 {
		/*
			fmt.Println(regexResult, len(regexResult))
			for i, r := range regexResult {
				fmt.Println(i, r)
			}
		*/
	} else if regexResult = MDYRegex.FindStringSubmatch(input); len(regexResult) == 3 {
		/*
			fmt.Println(regexResult, len(regexResult))
			for i, r := range regexResult {
				fmt.Println(i, r)
			}
		*/
	} else {
		//No match, proceed to next processed slide
		//fmt.Println("no regex match")
		//fmt.Println(input)
		noMatchDateHeaderInputs = append(noMatchDateHeaderInputs, input)
		return
	}

	var capturedYear int
	var capturedDay int
	if capturedYear, err = strconv.Atoi(regexResult[2]); err != nil {
		return
	}
	if capturedDay, err = strconv.Atoi(regexResult[1]); err != nil {
		return
	}

	//If found date is closer to time.Now than other dates, keep it and check other processed slides for other date matches
	date = time.Date(capturedYear, estimatedMonth, capturedDay, 0, 0, 0, 0, time.UTC)
	return
}

func closerDate(target time.Time, one time.Time, two time.Time) (closerDate time.Time) {
	if math.Abs(float64(target.Sub(one))) < math.Abs(float64(target.Sub(two))) {
		closerDate = one
	} else {
		closerDate = two
	}
	return
}