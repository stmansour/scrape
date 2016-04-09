csvbld: *.go
	go vet
	golint
	go build
	@echo "build complete in csvbld"

clean:
	rm -f csvbld
	@echo "clean complete in csvbld"

install:
	cp csvbld ../bin/
	@echo "install complete in csvbld"
