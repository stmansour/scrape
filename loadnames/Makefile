loadnames: *.go
	go vet
	golint
	go build
	@echo "build complete in loadnames"

clean:
	rm -f loadnames
	@echo "clean complete in loadnames"

install:
	cp loadnames ../bin/
	@echo "install complete in loadnames"
