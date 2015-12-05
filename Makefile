all: scrape

clean:
	cd csvbld;make clean
	cd form;make clean
	cd splitName;make clean
	rm -f scrape

scrape: *.go
	go build
