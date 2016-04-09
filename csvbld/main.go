package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"rentroll/rlib"
	"strings"
	"time"
)

// App is the global application structure
var App struct {
	htmldir   string // directory where we store html files
	fname     string // name of csv file to open and process
	startName string // skip over names in csv until finding this name
	skip      bool
}

func updatePerson(f, l string) {

}

func html2csv(fname string) {
	args := []string{"./html2csv.py", fname}
	cmd := exec.Command("python", args...)
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	cmd.Wait()
}

func loadProfile(url string) {
	randfname := fmt.Sprintf("%d", time.Now().UnixNano())
	htmlfname := randfname + ".html"
	resp, err := http.Get(url)
	rlib.Errcheck(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	rlib.Errcheck(err)
	err = ioutil.WriteFile(htmlfname, body, 0666)
	rlib.Errcheck(err)
	html2csv(htmlfname)
	rlib.Errcheck(os.Remove(htmlfname))
}

//-----------------------------------------------------------------------------------
// processLoadPerson takes as input strings that look like this:
//		Zuber, Jody; (LoadPerson)?OpenAgent&C20933A8A369FCFE85256EE400175DB2
// The mission is to build a URL that looks like urlbase + "(LoadPerson)?..."
// read back the profile information, which comes in html format. Then store it
// with the information we already have for the name to the left if the semicolon ;
// If no name is found, it can still be processed because the profile has the name
// embedded.
//-----------------------------------------------------------------------------------
func processLoadPerson(s string) {
	//-------------------------------------------------
	// Build the URL
	//-------------------------------------------------
	urlbase := "https://directory.faa.gov/appsPub/National/employeedirectory/faadir.nsf/"
	sa := strings.Split(s, ";")
	url := urlbase + strings.TrimSpace(sa[1])
	fmt.Printf("url = %s\n", url)

	//-------------------------------------------------
	// Build the name...
	//-------------------------------------------------
	firstName := ""
	lastName := ""
	if len(sa[0]) > 0 {
		// fmt.Printf("sa[0] = '%s'\n", sa[0])
		na := strings.Split(sa[0], ",")
		// fmt.Printf("na = %#v\n", na)
		firstName = strings.TrimSpace(na[1])
		lastName = strings.TrimSpace(na[0])
	}

	//-------------------------------------------------
	// Since the list is large (>60,000 people) we may
	// encounter errors that break the program. If so,
	// we may want to restart at a particular name rather
	// than starting from scratch each time.
	//-------------------------------------------------
	if App.skip {
		if App.startName == lastName {
			fmt.Printf("found %s, will begin processing now\n", lastName)
			App.skip = false
		} else {
			return
		}
	}

	//-------------------------------------------------
	// Get the info and update...
	//-------------------------------------------------
	loadProfile(url)
	updatePerson(firstName, lastName)
}

func readCommandLineArgs() {
	dirPtr := flag.String("d", ".", "directory to store html files")
	fPtr := flag.String("f", "step4.csv", "csv file to load")
	skipPtr := flag.String("s", "", "skip input lines until finding this name.")
	flag.Parse()
	App.htmldir = *dirPtr
	App.fname = *fPtr
	App.startName = *skipPtr
}

func main() {
	readCommandLineArgs()
	f, err := os.Open(App.fname)
	rlib.Errcheck(err)
	defer f.Close()

	App.skip = false
	if len(App.startName) > 0 {
		fmt.Printf("Skipping to: %s\n", App.startName)
		App.skip = true
	}

	//-------------------------------------------------
	// Read each line and send it to be processed...
	//-------------------------------------------------
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		processLoadPerson(s)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
