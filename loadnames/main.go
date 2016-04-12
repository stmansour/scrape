// auser  a program to set the role for user in the accord database
//        based on their UID
package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
)

import _ "github.com/go-sql-driver/mysql"

// Person is a structure of all attributes of the FAA employees we're capturing
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
}

// collection of prepared sql statements
type prepSQL struct {
	insertPerson *sql.Stmt
}

// App is the global data structure for this app
var App struct {
	db        *sql.DB
	DBName    string
	DBUser    string
	fname     string
	startName string
	prepstmt  prepSQL
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

// InsertPerson writes a new Person record to the database
func InsertPerson(p *Person) error {
	_, err := App.prepstmt.insertPerson.Exec(
		p.FirstName, p.LastName, p.MiddleName, p.JobTitle, p.OfficePhone, p.OfficeFax, p.Email1, p.MailAddress, p.MailAddress2, p.MailCity, p.MailState, p.MailPostalCode, p.MailCountry, p.RoomNumber, p.MailStop)
	if nil != err {
		fmt.Printf("error inserting person: %v\n", err)
	}
	return err
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
	return stripchars(s, " ,'\"():;<>")
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
}

func nameHandler(s string, p *Person) {
	// first, split last and first
	sa := strings.Split(s, ",")
	l := len(sa)
	for i := 0; i < l; i++ {
		sa[i] = strings.TrimSpace(sa[i])
	}

	// see if there is anything extra in the first name that we can
	// use as a middle name or initial
	if l == 2 {
		ta := strings.Split(sa[1], " ")
		if len(ta) > 1 {
			sa[1] = ta[0]
			sa = append(sa, ta[1])
			l = len(sa)
		}
	}
	switch {
	case l == 3:
		p.MiddleName = strings.TrimSpace(sa[2])
		fallthrough
	case l == 2:
		p.LastName = strings.TrimSpace(sa[0])
		p.FirstName = strings.TrimSpace(sa[1])
	case l == 1:
		p.LastName = strings.TrimSpace(sa[0])
	default:
		fmt.Printf("unknown format: sa = %#v\n", sa)
	}
}

func loadnames(fname string) {
	f, err := os.Open(fname)
	Errcheck(err)
	defer f.Close()
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	rawCSVdata, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, sa := range rawCSVdata {
		// lines are in the following format:  name, jobtitle, officephone, profile, orgchard
		//    "Aakre, Dave C","ATSS","701-451-6805"," View Profile","N/A"
		// profile and orgchart are just text in this file (http links removed), so ignore them
		if 5 != len(sa) {
			fmt.Printf("Number of fields is not 5 for sa: %#v\n", sa)
			return
		}
		var p Person
		p.JobTitle = sa[1]
		p.OfficePhone = sa[2]
		nameHandler(sa[0], &p)
		emailBuilder(&p)
		InsertPerson(&p)
	}
}

func buildPreparedStatements() {
	var err error
	App.prepstmt.insertPerson, err = App.db.Prepare("INSERT INTO people (FirstName,LastName,MiddleName,JobTitle,OfficePhone,OfficeFax,Email1,MailAddress,MailAddress2,MailCity,MailState,MailPostalCode,MailCountry,RoomNumber,MailStop) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	Errcheck(err)
}

func readCommandLineArgs() {
	dbuPtr := flag.String("B", "ec2-user", "database user name")
	dbnmPtr := flag.String("N", "faa", "database name")
	fPtr := flag.String("f", "step3.csv", "name of csvfile to parse")
	sPtr := flag.String("s", "", "skip names until you find this name, then engage")
	flag.Parse()
	App.DBName = *dbnmPtr
	App.DBUser = *dbuPtr
	App.fname = *fPtr
	App.startName = *sPtr
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
