package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func errcheck(err error) {
	if err != nil {
		fmt.Printf("err = %v\n", err)
		os.Exit(1)
	}
}

func grabProfile(url string) {
	args := []string{url}
	cmd := exec.Command("./extract.sh", args...)
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	cmd.Wait()
}

func strToInt(s string) int {
	if len(s) == 0 {
		return 0
	}
	s = strings.Trim(s, " \n\r")
	n, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("Error converting %s to a number: %v\n", s, err)
		return 0
	}
	return n
}

// processProfile extracts the useful info:
// lines 14-18 contain address info, randomly dispersed
// line 19 contains room number
// line 20 contains mail stop

var address []string
var room string
var mailstop string
var city string
var state string
var zip string
var digitsRegexp = regexp.MustCompile(`\d+`)
var name string
var fcsv *os.File

func processProfile() {
	file, err := os.Open("profile.html")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	row := 0
	lc4 := len("<COL:4>")
	address = make([]string, 0)
	room = ""
	mailstop = ""

	for scanner.Scan() {
		s := scanner.Text()
		if strings.Contains(s, "[ROW:") {
			row = strToInt(digitsRegexp.FindString(s))
		} else {
			if row < 14 && row > 20 {
				continue
			}
			i := strings.Index(s, "<COL:4>")
			if i < 0 {
				continue
			}
			s1 := s[i+lc4:]
			if len(s1) == 0 || strings.Contains(s1, "/icons/ecblank.gif") {
				continue
			}
			fmt.Printf("row%d:  %s\n", row, s1)
			switch {
			case 14 <= row && row <= 18:
				address = append(address, s1)
			case row == 19:
				room = s1
			case row == 20:
				mailstop = s1
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func parseCityStateZip() {
	city = ""
	state = ""
	zip = ""
	if len(address) == 0 {
		return
	}
	s := address[len(address)-1]
	i := strings.Index(s, ",")
	if i < 0 {
		return
	}
	city = s[:i]
	l := len(s)
	zip = s[l-5:]
	state = s[i+2 : l-6]
}

//  dumpAddress formats the address, room, and mailstop as follows:
//         "addr1", "addr2", "addr3", "addr4", "city", "state", "zip"
func dumpAddress() {
	s := ""
	for i := 1; i < 5-len(address); i++ {
		s += fmt.Sprintf("\"\",")
	}
	for i := 0; i < len(address)-1; i++ {
		s += fmt.Sprintf("\"%s\",", address[i])
	}
	s += fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"", city, state, zip, room, mailstop)

	out, err := exec.Command("grep", name, "../faa.csv").Output()
	errcheck(err)
	if len(out) > 0 {

		rsp := string(out)
		rsp = strings.TrimSpace(rsp)
		fmt.Fprintf(fcsv, "%s,%s\n", rsp, s)
	}
}

func main() {
	urlbase := "https://directory.faa.gov/appsPub/National/employeedirectory/faadir.nsf/"
	if len(os.Args) < 2 {
		fmt.Printf("You must supply the filename on the command line\n")
		os.Exit(1)
	}
	filename := os.Args[1]
	f, err := os.Open(filename)
	errcheck(err)
	defer f.Close()

	startName := ""
	if len(os.Args) == 3 {
		startName = os.Args[2]
		fmt.Println("Skipping to: %s\n", startName)
	}

	fcsv, err = os.OpenFile("final.csv", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	errcheck(err)
	defer fcsv.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	rawCSVdata, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// sanity check, display to standard output
	skip := true
	if startName == "" {
		skip = false
	}

	for _, da := range rawCSVdata {
		url := fmt.Sprintf("%s%s", urlbase, da[1])
		name = da[0]

		if skip {
			if startName == name {
				fmt.Printf("found %s, will begin processing now\n", name)
				skip = false
			} else {
				fmt.Printf("%s != %s\n", name, startName)
				continue
			}
		}
		fmt.Printf("name=\"%s\", url=\"%s\"\n", name, url)
		grabProfile(url)
		processProfile()

		// fmt.Printf("name: %s\naddress: %v\nroom: %s\nmailstop: %s\n", da[0], address, room, mailstop)

		parseCityStateZip()
		fmt.Printf("city: %s     state: %s      zip: %s\n", city, state, zip)

		dumpAddress()
	}
}
