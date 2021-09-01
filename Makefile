PROJECTNAME=$(shell basename "$(PWD)")

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: clean
clean:
	@echo " > Cleaning build cache..."
	go clean
	rm -f bin/greenplum_exporter_linux
	rm -f bin/greenplum_exporter_mac
	rm -f bin/greenplum_exporter_win
	rm -fr bin/dist

.PHONY: build
build:
	@echo " > Building binary..."
	if [ ! -d bin/ ]; then mkdir bin/ ; fi;
	go mod download && go build -o ./bin/greenplum_exporter_mac
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/greenplum_exporter_linux
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/greenplum_exporter_win

.PHONY: package
package:
	@echo " > Archive binary target files and srcipts..."
	if [ ! -d bin/ ]; then mkdir bin/ ; fi;
	cd bin/ && mkdir -p dist && mkdir -p tmp && cd -
	go mod download && go build -o ./bin/tmp/greenplum_exporter_mac
	go build -o ./bin/tmp/greenplum_exporter_linux
	go build -o ./bin/tmp/greenplum_exporter_win
	cd bin/tmp/ && tar -czvf ../dist/greenplum_exporter.tgz * && cd -
	cd bin/ && rm -fr tmp/ && cd -
