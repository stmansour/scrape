DIRS = form html2csv csvbld profile db loadnames checker

scrape:
	for dir in $(DIRS); do make -C $$dir;done

clean:
	for dir in $(DIRS); do make -C $$dir clean;done

install:
	if [ -d ./bin ]; then rm -rf ./bin; fi
	mkdir ./bin
	for dir in $(DIRS); do make -C $$dir install;done

