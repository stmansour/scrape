package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"rentroll/rlib"
	"runtime/debug"
	"strings"
)

import _ "github.com/go-sql-driver/mysql"

// collection of prepared sql statements
type prepSQL struct {
	getPersonByName  *sql.Stmt
	getPersonByName2 *sql.Stmt
	getPersonByEmail *sql.Stmt
	updatePerson     *sql.Stmt
}

// App is the global data structure for this app
var App struct {
	db       *sql.DB
	DBName   string
	DBUser   string
	fname    string
	prepstmt prepSQL
	debug    bool
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

// Errcheck - saves a bunch of typing, prints error if it exists
//            and provides a traceback as well
func Errcheck(err error) {
	if err != nil {
		fmt.Printf("error = %v\n", err)
		debug.PrintStack()
		log.Fatal(err)
	}
}

// debug logger
func dlog(format string, a ...interface{}) {
	p := fmt.Sprintf(format, a...)
	if App.debug {
		fmt.Print(p)
	}
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
	} else {
		// fmt.Printf("Get person by first, last:  \"%s\", \"%s\"\n", first, last)
		err = App.prepstmt.getPersonByName2.QueryRow(first, last).Scan(&p.FID,
			&p.FirstName, &p.LastName, &p.MiddleName, &p.JobTitle, &p.OfficePhone, &p.OfficeFax,
			&p.Email1, &p.MailAddress, &p.MailAddress2, &p.MailCity, &p.MailState, &p.MailPostalCode,
			&p.MailCountry, &p.RoomNumber, &p.MailStop)
		if nil != err {
			// fmt.Printf("GetPersonByName(%s,%s): error = %#v\n", first, last, err)
		}
	}
	return p
}

// GetAllPeopleWithName reads an array of Person structs for all records that match the supplied first and last name.
func GetAllPeopleWithName(first, last string) []Person {
	var t []Person
	var err error
	rows, err := App.prepstmt.getPersonByName2.Query(first, last)
	rlib.Errcheck(err)
	defer rows.Close()
	for rows.Next() {
		var p Person
		rows.Scan(&p.FID,
			&p.FirstName, &p.LastName, &p.MiddleName, &p.JobTitle, &p.OfficePhone, &p.OfficeFax,
			&p.Email1, &p.MailAddress, &p.MailAddress2, &p.MailCity, &p.MailState, &p.MailPostalCode,
			&p.MailCountry, &p.RoomNumber, &p.MailStop)
		if nil != err {
			//fmt.Printf("GetAllPeopleWithName(%s,%s): error = %#v\n", first, last, err)
		} else {
			t = append(t, p)
		}
	}
	return t
}

// GetPersonByEmail reads a person record from the database where the email matches the supplied email
func GetPersonByEmail(email string) Person {
	var p Person
	var err error

	err = App.prepstmt.getPersonByEmail.QueryRow(email).Scan(&p.FID,
		&p.FirstName, &p.LastName, &p.MiddleName, &p.JobTitle, &p.OfficePhone, &p.OfficeFax,
		&p.Email1, &p.MailAddress, &p.MailAddress2, &p.MailCity, &p.MailState, &p.MailPostalCode,
		&p.MailCountry, &p.RoomNumber, &p.MailStop)
	dlog("GetPersonByEmail(%s): error = %#v\n", email, err)
	dlog("GetPersonByEmail ->  p.FID = %d\n", p.FID)
	return p
}

func loadCSV(fname string) [][]string {
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

func readCommandLineArgs() {
	dbuPtr := flag.String("B", "ec2-user", "database user name")
	dbnmPtr := flag.String("N", "faa", "database name")
	fPtr := flag.String("f", "validatedFAAeMailAddr.csv", "name of csvfile to parse")
	dbgPtr := flag.Bool("D", false, "debug mode")
	// sPtr := flag.String("s", "", "skip names until you find this name, then engage")
	flag.Parse()
	App.DBName = *dbnmPtr
	App.DBUser = *dbuPtr
	App.fname = *fPtr
	App.debug = *dbgPtr
	// App.startName = *sPtr
}

func buildPreparedStatements() {
	var err error
	App.prepstmt.getPersonByName, err = App.db.Prepare("SELECT * FROM people WHERE FirstName=? and MiddleName=? and LastName=?")
	rlib.Errcheck(err)
	App.prepstmt.getPersonByName2, err = App.db.Prepare("SELECT * FROM people WHERE FirstName=? and LastName=?")
	rlib.Errcheck(err)
	App.prepstmt.getPersonByEmail, err = App.db.Prepare("SELECT * FROM people WHERE Email1=?")
	rlib.Errcheck(err)
	App.prepstmt.updatePerson, err = App.db.Prepare("UPDATE people SET FirstName=?,LastName=?,MiddleName=?,JobTitle=?,OfficePhone=?,OfficeFax=?,Email1=?,MailAddress=?,MailAddress2=?,MailCity=?,MailState=?,MailPostalCode=?,MailCountry=?,RoomNumber=?,MailStop=? where FID=?")
	rlib.Errcheck(err)
}

func getPersonNameString(p *Person) string {
	pn := p.FirstName
	if len(p.MiddleName) > 0 {
		pn += " " + p.MiddleName
	}
	pn += " " + p.LastName
	return pn
}

func possibleNameUpdate(first, last, email string, p *Person) {
	fmt.Printf("POSSIBLE NAME UPDATE\n")
	fmt.Printf("           Guest data: %s %s (%s)\n", first, last, email)
	fmt.Printf("    Possible DB match: %s (FID=%d, %s)\n", getPersonNameString(p), p.FID, p.Email1)
	fmt.Printf("\n")
}

func possibleEmailUpdate(first, last, email string, tp *[]Person) {
	fmt.Printf("POSSIBLE EMAIL UPDATE\n")
	fmt.Printf("           Guest data: %s %s (%s)\n", first, last, email)
	fmt.Printf("  Possible db matches: ")
	fmt.Printf("%s (FID=%d, %s)\n", getPersonNameString(&(*tp)[0]), (*tp)[0].FID, (*tp)[0].Email1)
	for i := 1; i < len(*tp); i++ {
		fmt.Printf("                       %s (FID=%d, %s)\n", getPersonNameString(&(*tp)[i]), (*tp)[i].FID, (*tp)[i].Email1)
	}
	fmt.Printf("\n")
}

func loadnames(fname string) {
	potentialEmailUpdate := 0
	matches := 0
	potentialNameUpdate := 0
	namesNotFound := 0

	// strings are of the form:
	//  "Agcaoili, Michael","michael.agcaoili@faa.gov"
	t := loadCSV(fname)

	for i := 0; i < len(t); i++ {
		sa := t[i]                        // name = sa[0], email = sa[1]
		na := strings.Split(sa[0], ",")   // "Agcaoili" " Michael"
		first := strings.TrimSpace(na[1]) // "Michael"
		last := strings.TrimSpace(na[0])  // "Agcaoili"
		email := strings.TrimSpace(sa[1]) // "ichael.agcaoili@faa.gov"

		// handle the situation where the first name was entered as First Middle...
		fa := strings.Split(first, " ")
		if len(fa) > 0 {
			first = fa[0]
		}

		// Can we find this person?
		t := GetAllPeopleWithName(first, last)
		dlog("%d matches for firstname = %s, lastname = %s\n", len(t), first, last)
		if len(t) > 0 {
			dlog("Found %d people with name %s %s\n", len(t), first, last)
			found := false
			for i := 0; !found && i < len(t); i++ {
				dlog("person[%d].Email1 = %s, looking for %s\n", i, t[i].Email1, email)
				if strings.ToLower(email) != strings.ToLower(t[i].Email1) {
					dlog("MISMATCH\n")
					// dlog("Email address mismatch: %s %s  DB email = %s, provided email = %s\n",
					// first, last, t[i].Email1, email)
					continue // there may be other matches on the name
				} else {
					matches++
					found = true
				}
			}
			if !found { // did we match a name and an email address?
				// do we have the email address under a different name?
				dlog("Looking for any record with email address %s\n", email)
				p := GetPersonByEmail(email)
				if p.FID > 0 {
					possibleNameUpdate(first, last, email, &p)
					potentialNameUpdate++
				} else {
					possibleEmailUpdate(first, last, email, &t)
					potentialEmailUpdate++
				}
			}
		} else {
			// do we have the email address under a different name?
			p := GetPersonByEmail(email)
			dlog("Looking for any record with email address %s\n", email)
			if p.FID > 0 {
				possibleNameUpdate(first, last, email, &p)
				potentialNameUpdate++
			} else {
				fmt.Printf("NOT FOUND IN DB: %s %s or %s\n\n", first, last, email)
				namesNotFound++
			}
		}
	}

	fmt.Printf("-----------------------------------------------------------\n")
	fmt.Printf("Matches: %d\n", matches)
	fmt.Printf("Possible name updates: %d\n", potentialNameUpdate)
	fmt.Printf("Possible email updates: %d\n", potentialEmailUpdate)
	fmt.Printf("Names and email addresses that could not be found: %d\n", namesNotFound)
	fmt.Printf("Total entries processed: %d\n", len(t))
}

func main() {
	readCommandLineArgs()

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
	loadnames(App.fname)
}
