
deployment: clean build zip

clean:
	rm -f lambda.zip
	rm -f handler

build:
	go build -o handler main.go

zip:
	zip lambda.zip handler

