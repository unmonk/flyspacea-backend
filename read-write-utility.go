package main

import (
	"encoding/json"
	"github.com/bradfitz/latlong"
	"io/ioutil"
	"time"
	"fmt"
	//"log"
)

func (t *Terminal) getTZ() (err error) {
	tz := latlong.LookupZoneName(
			(*t).Location.Latitude,
			(*t).Location.Longitude)

	if len(tz) == 0 {
		err = fmt.Errorf("latlong.LookupZoneName returned empty string for Terminal %v\n", (*t))
	}

	(*t).Timezone, err = time.LoadLocation(
		tz)
	return
}

func readTerminalArrayToMap(terminalArray []Terminal) (terminalMap map[string]Terminal) {
	//set key to v.Title and set v.Index
	terminalMap = make(map[string]Terminal)
	for _, v := range terminalArray {
		terminalMap[v.Title] = v
	}

	return
}

func readTerminalArrayFromFiles(filenames ...string) (keywordsArray []Terminal, err error) {
	for _, filename := range filenames {
		var locationsRaw []byte
		locationsRaw, err = ioutil.ReadFile(filename)
		if err != nil {
			return
		}

		var tmp []Terminal
		if err = json.Unmarshal(locationsRaw, &tmp); err != nil {
			return
		}

		//Get Timezone for every terminal
		for i, _ := range tmp {
			if err = tmp[i].getTZ(); err != nil {
				return
			}
			fmt.Println(tmp[i].Timezone, tmp[i].Title, tmp[i].Location)
		}

		//log.Printf("%v locations in file %v", len(tmp), filenames)

		keywordsArray = append(keywordsArray, tmp...)
	}

	return
}
