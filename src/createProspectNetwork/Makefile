OUTFILE=lambda
PACKAGE=lambda.zip
deployment: clean build zip

clean:
	rm -f $(PACKAGE)
	rm -f $(OUTFILE)

build:
	go build -o $(OUTFILE) main.go

zip:
	zip $(PACKAGE) $(OUTFILE)

