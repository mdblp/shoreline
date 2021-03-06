#!/bin/sh
go get -u github.com/jstemmer/go-junit-report
go get github.com/t-yuki/gocover-cobertura
go test -v -race -coverprofile=coverage.out ./... 2>&1 > testresults.txt
testPass=$?
cat testresults.txt
cat testresults.txt | go-junit-report  > test-report.xml
if [ $testPass -eq 1 ]; then
  echo "Test failled"
  exit 1
fi
gocover-cobertura < coverage.out > coverage.xml
go tool cover -html='coverage.out' -o coverReport.html
