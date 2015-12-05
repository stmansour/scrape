package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

var fcsv *os.File

func errcheck(err error) {
	if err != nil {
		fmt.Printf("err = %v\n", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("You must supply the filename on the command line\n")
		os.Exit(1)
	}
	filename := os.Args[1]
	f, err := os.Open(filename)
	errcheck(err)
	defer f.Close()

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

	for _, da := range rawCSVdata {
		n := strings.Split(da[0], ",")

		for i := 1; i < len(da); i++ {
			n = append(n, da[i])
		}

		s := fmt.Sprintf("\"%s\"", n[0])
		for i := 1; i < len(n); i++ {
			s += fmt.Sprintf(",\"%s\"", n[i])
		}
		fmt.Fprintf(fcsv, "%s\n", s)
	}
}
