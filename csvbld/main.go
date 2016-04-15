package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"rentroll/rlib"
	"strings"
	"sync"
	"time"
)

import _ "github.com/go-sql-driver/mysql"

// App is the global application structure
var App struct {
	htmldir   string // directory where we store html files
	fname     string // name of csv file to open and process
	startName string // skip over names in csv until finding this name
	skip      bool
	db        *sql.DB
	DBName    string
	DBUser    string
	binpath   string
	prepstmt  prepSQL
	c         chan string
	workers   int // number of workers in the goroutine worker pool
	debug     bool
	quick     bool // only go through one loop
}

// Person is the structure that defines all the attributes of a person
type Person struct {
	FID            int64
	FirstName      string
	LastName       string
	MiddleName     string
	JobTitle       string
	OfficePhone    string
	OfficeFax      string
	Email1         string
	MailAddress    string
	MailAddress2   string
	MailCity       string
	MailState      string
	MailPostalCode string
	MailCountry    string
	RoomNumber     string
	MailStop       string
	PreferredName  string
}

// collection of prepared sql statements
type prepSQL struct {
	getPersonByName  *sql.Stmt
	getPersonByName2 *sql.Stmt
	updatePerson     *sql.Stmt
}

// GetPersonByName reads a person record from the database where the name matches the supplied name
func GetPersonByName(first, middle, last string) Person {
	var p Person
	var err error
	if len(middle) > 0 {
		// fmt.Printf("Get person by first, middle, last:  \"%s\", \"%s, \"%s\"\n", first, middle, last)
		err = App.prepstmt.getPersonByName.QueryRow(first, middle, last).Scan(&p.FID,
			&p.FirstName, &p.LastName, &p.MiddleName, &p.JobTitle, &p.OfficePhone, &p.OfficeFax,
			&p.Email1, &p.MailAddress, &p.MailAddress2, &p.MailCity, &p.MailState, &p.MailPostalCode,
			&p.MailCountry, &p.RoomNumber, &p.MailStop)
		if nil == err {
			return p
		}
	}
	if nil != err {
		if false == strings.Contains(err.Error(), "sql: no rows") {
			fmt.Printf("GetPersonByName(%s,%s,%s): error = %#v\n", first, middle, last, err)
			return p
		}
		// fmt.Printf("First,middle,Last was in the profile, but not in the database. Trying first, last.\n")
	}
	// fmt.Printf("Get person by first, last:  \"%s\", \"%s\"\n", first, last)
	err = App.prepstmt.getPersonByName2.QueryRow(first, last).Scan(&p.FID,
		&p.FirstName, &p.LastName, &p.MiddleName, &p.JobTitle, &p.OfficePhone, &p.OfficeFax,
		&p.Email1, &p.MailAddress, &p.MailAddress2, &p.MailCity, &p.MailState, &p.MailPostalCode,
		&p.MailCountry, &p.RoomNumber, &p.MailStop)
	if nil != err {
		fmt.Printf("GetPersonByName(%s,%s): error = %#v\n", first, last, err)
	}
	return p
}

// DBUpdatePerson updates the existing database record for p with the information in p
func DBUpdatePerson(p *Person) {
	if len(p.MailState) > 10 {
		fmt.Printf("MailState is too large. person = %#v\n", *p)
		err := fmt.Errorf("State is too big: %s", p.MailState)
		rlib.Errcheck(err)
	}
	if App.debug {
		fmt.Printf("Insert person:  %#v\n", *p)
	}
	_, err := App.prepstmt.updatePerson.Exec(p.FirstName, p.LastName, p.MiddleName, p.JobTitle, p.OfficePhone, p.OfficeFax, p.Email1, p.MailAddress, p.MailAddress2, p.MailCity, p.MailState, p.MailPostalCode, p.MailCountry, p.RoomNumber, p.MailStop, p.FID)
	rlib.Errcheck(err)
}

func html2csv(fname string) {
	h := fmt.Sprintf("%s/html2csv.py", App.binpath)
	args := []string{h, fname}
	cmd := exec.Command("python", args...)
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	cmd.Wait()
}

func loadProfileCSV(fname string) [][]string {
	t := [][]string{}
	f, err := os.Open(fname)
	rlib.Errcheck(err)
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	rawCSVdata, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, sa := range rawCSVdata {
		t = append(t, sa)
	}
	return t
}

func stripchars(str, chr string) string {
	return strings.Map(func(r rune) rune {
		if strings.IndexRune(chr, r) < 0 {
			return r
		}
		return -1
	}, str)
}

func scrubEmailAddr(s string) string {
	return stripchars(s, " ,\"():;<>")
}

// emailBuilder generates an email address based on the apparent
// default formula that the FAA uses for their email addresses.
// That is:
//		[firstName].[lastName]@FAA.gov
// or
//		[firstName].[middleInitial].[lastName]@FAA.gov
func emailBuilder(p *Person) {
	if len(p.MiddleName) > 0 {
		p.Email1 = scrubEmailAddr(fmt.Sprintf("%s.%s.%s@faa.gov", p.FirstName, p.MiddleName, p.LastName))
	} else if len(p.FirstName) > 0 {
		p.Email1 = scrubEmailAddr(fmt.Sprintf("%s.%s@faa.gov", p.FirstName, p.LastName))
	}
	fmt.Printf("set email address to: %s\n", p.Email1)
}

func badAddress(s string, p *Person) {
	fmt.Printf("Person = %#v\n", p)
	if len(s) == 0 {
		rlib.Errcheck(fmt.Errorf("Address string s:  len(s) == 0\n"))
	}
	rlib.Errcheck(fmt.Errorf("Unrecognized address format:  %s\n", s))
}

func parseCityStateZip(address []string, p *Person) {
	if len(address) == 0 {
		return
	}
	var s string
	s = strings.TrimSpace(address[len(address)-1])
	if len(s) == 0 {
		return
	}
	sa := strings.Split(s, ",")
	l := len(sa)
	if l == 2 {
		p.MailCity = strings.TrimSpace(sa[0])
		ta := strings.Split(sa[1], " ")
		if len(ta) > 1 {
			p.MailPostalCode = strings.TrimSpace(ta[len(ta)-1])
			p.MailState = strings.TrimSpace(strings.Join(ta[0:len(ta)-1], " "))
		}
	} else {
		// LOOK FOR KNOWN ERRONEOUS PATTERNS
		// Fort Worth, TX 76177, TX 76177
		if l == 3 {
			if strings.TrimSpace(sa[1]) == strings.TrimSpace(sa[2]) {
				p.MailCity = strings.TrimSpace(sa[0])
				ta := strings.Split(sa[1], " ")
				if len(ta) > 1 {
					p.MailPostalCode = strings.TrimSpace(ta[len(ta)-1])
					p.MailState = strings.TrimSpace(strings.Join(ta[0:len(ta)-1], " "))
				}
			} else {
				badAddress(s, p)
			}
		} else {
			badAddress(s, p)
		}
	}
	p.MailAddress = address[len(address)-2]

	// fmt.Printf("Mail Address:    %s\n", p.MailAddress)
	// fmt.Printf("Mail City:       %s\n", p.MailCity)
	// fmt.Printf("Mail State:      %s\n", p.MailState)
	// fmt.Printf("Mail PostalCode: %s\n", p.MailPostalCode)

	return
}

func debugDumpProfileCSV(tp *[][]string) {
	//quick debug
	t := *tp
	for i := 0; i < len(t); i++ {
		for j := 0; j < len(t[i]); j++ {
			fmt.Printf("t[%d][%d] = %s\n", i, j, t[i][j])
		}
	}
}

func loadProfile(url string, firstName, lastName string) {
	randfname := fmt.Sprintf("%d", time.Now().UnixNano())
	htmlfname := randfname + ".html"

	// let's do retries here... 3 tries.  Wait 5 seconds between each try...
	var err error
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = http.Get(url)
		if nil == err {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		fmt.Printf("http.Get(%s) failed 3 times\n", url)
		fmt.Printf("\terr = %v\n", err)
		return // let's let the program keep running
	}

	defer resp.Body.Close()
	var body []byte
	for i := 0; i < 3; i++ {
		body, err := ioutil.ReadAll(resp.Body)
		if nil == err {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		fmt.Printf("loadProfile: ioutil.ReadAll failed 3 times\n")
		fmt.Printf("\terr = %v\n", err)
		return // let's let the program keep running
	}

	err = ioutil.WriteFile(htmlfname, body, 0666)
	rlib.Errcheck(err)
	html2csv(htmlfname)
	rlib.Errcheck(os.Remove(htmlfname))

	t := loadProfileCSV(randfname + ".csv")
	if App.debug {
		debugDumpProfileCSV(&t)
	}

	// note that the first name (A) in the line containing the profile address
	// may be different than the first name in the profile (B).  The database entry
	// is created based on name A, we may need to update the record with the name
	// in B.
	sa := strings.Split(firstName, " ")
	middleName := ""
	if len(sa) > 1 {
		firstName = sa[0]
		middleName = sa[1]
	}

	// parse the name...
	first := ""
	middle := ""
	last := ""
	if len(t) > 2 {
		name := t[2][0]
		if len(name) > 0 {
			na := strings.Split(name, " ")
			l := len(na)
			switch {
			case l == 2:
				first = na[0]
				last = na[1]
			case l == 3:
				first = na[0]
				middle = na[1]
				last = na[2]
			default:
				fmt.Printf("unrecognized name format: %#v\n", na)
				rlib.Errcheck(os.Remove(randfname + ".csv"))
				return
			}
		}
		// fmt.Printf("Found name.  first(%s)  middle(%s)  last(%s)\n", first, middle, last)

		// load the database record for this person
		p := GetPersonByName(firstName, middleName, lastName) // try name A
		if p.FID == 0 && len(first) > 0 && len(last) > 0 {
			p = GetPersonByName(first, middle, last)
			if p.FID == 0 {
				fmt.Printf("INFO: Could not find person named:  %s %s %s\n", firstName, middleName, lastName)
				rlib.Errcheck(os.Remove(randfname + ".csv"))
				return
			}
		}

		// Update the address.  The address is contained in the array of strings between
		// rows 14-17 in column 3.   that is [14..17][3]
		var addr []string
		for i := 14; i < 18; i++ { // these are the rows in which an address MIGHT appear
			if len(t[i]) >= 4 { // if there are 4 columns
				if len(t[i][3]) > 0 { // if there's anything in the column...
					addr = append(addr, t[i][3]) // grab it
				}
			}
		}

		// room number is at row 18. Mailstop is row 19
		p.RoomNumber = strings.TrimSpace(t[18][3])
		p.MailStop = strings.TrimSpace(t[19][3])

		// check for a different first name in the profile compared to the
		// first name that came from the search results
		if first != firstName {
			fmt.Printf("INFO:  Names differ: [%s %s %s]  vs  [%s %s %s]\n", first, middle, last, firstName, middleName, lastName)
			// if different, we use the simple assumption that the longer of
			// the names is the proper name, and the other name is the PreferredName
			if len(firstName) < len(first) {
				emailBuilder(&p)
				p.FirstName = first
				p.PreferredName = firstName
			}
		}
		parseCityStateZip(addr, &p)
		DBUpdatePerson(&p)
	} else {
		fmt.Printf("INFO:  len(t) < 3. Person = %s %s. url = %s", firstName, lastName, url)
	}
	rlib.Errcheck(os.Remove(randfname + ".csv"))
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
	// Get the info and update...
	//-------------------------------------------------
	fmt.Printf("start: %s %s\n", firstName, lastName)
	loadProfile(url, firstName, lastName)
	fmt.Printf("done: %s %s\n", firstName, lastName)
}

// worker waits for a work item (string s) to come to it via the
// channel string. When it gets one, it calls processLoadPerson to
// handle that string. It will continue doing this as long as more
// work is available via channel n.  Once n is closed, it will exit
// which invokes the deferred work group exit.
func worker(n chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for s := range n {
		processLoadPerson(s)
	}
}

func buildPreparedStatements() {
	var err error
	App.prepstmt.getPersonByName, err = App.db.Prepare("SELECT * FROM people WHERE FirstName=? and MiddleName=? and LastName=?")
	rlib.Errcheck(err)
	App.prepstmt.getPersonByName2, err = App.db.Prepare("SELECT * FROM people WHERE FirstName=? and LastName=?")
	rlib.Errcheck(err)
	App.prepstmt.updatePerson, err = App.db.Prepare("UPDATE people SET FirstName=?,LastName=?,MiddleName=?,JobTitle=?,OfficePhone=?,OfficeFax=?,Email1=?,MailAddress=?,MailAddress2=?,MailCity=?,MailState=?,MailPostalCode=?,MailCountry=?,RoomNumber=?,MailStop=? where FID=?")
	rlib.Errcheck(err)
}

func readCommandLineArgs() {
	dirPtr := flag.String("d", ".", "directory to store html files")
	fPtr := flag.String("f", "step4.csv", "csv file to load")
	skipPtr := flag.String("s", "", "skip input lines until finding this name.")
	dbuPtr := flag.String("B", "ec2-user", "database user name")
	dbnmPtr := flag.String("N", "faa", "database name")
	binPtr := flag.String("b", ".", "path to bin, from current directory")
	dbgPtr := flag.Bool("D", false, "use this option to turn on debug mode")
	wpPtr := flag.Int("w", 25, "Number of workers in the worker pool")
	qPtr := flag.Bool("q", false, "quick option")
	flag.Parse()
	App.htmldir = *dirPtr
	App.fname = *fPtr
	App.startName = *skipPtr
	App.DBName = *dbnmPtr
	App.DBUser = *dbuPtr
	App.debug = *dbgPtr
	App.workers = *wpPtr
	App.binpath = *binPtr
	App.quick = *qPtr
}

func main() {
	start := time.Now()
	readCommandLineArgs()

	//------------------------------------------
	// Get the database open and ready for use
	//------------------------------------------
	var err error
	// s := "<awsdbusername>:<password>@tcp(<rdsinstancename>:3306)/accord"
	s := fmt.Sprintf("%s:@/%s?charset=utf8&parseTime=True", App.DBUser, App.DBName)
	App.db, err = sql.Open("mysql", s)
	if nil != err {
		fmt.Printf("sql.Open for database=%s, dbuser=%s: Error = %v\n", App.DBName, App.DBUser, err)
	}
	defer App.db.Close()
	err = App.db.Ping()
	if nil != err {
		fmt.Printf("App.db.Ping for database=%s, dbuser=%s: Error = %v\n", App.DBName, App.DBUser, err)
		os.Exit(1)
	}
	buildPreparedStatements()

	//------------------------------------------
	// Create a pool of worker goroutines...
	//------------------------------------------
	App.c = make(chan string)
	wg := new(sync.WaitGroup)
	for i := 0; i < App.workers; i++ {
		wg.Add(1)
		go worker(App.c, wg)
	}

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
		s := scanner.Text() // grab a new line of work
		if App.skip {
			if strings.Contains(s, App.startName) {
				fmt.Printf("Found %s.  Processing begins...\n", App.startName)
				App.skip = false
			} else {
				continue
			}
		}
		App.c <- s // hand it off to a worker
	}
	rlib.Errcheck(scanner.Err())
	close(App.c)

	// now just wait for the workers to finish everything...
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Elapsed time: %s\n", elapsed)
}
