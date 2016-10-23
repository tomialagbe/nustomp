GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOINSTALL=$(GOCMD) install
GOTEST=$(GOCMD) test
#GODEP=$(GOTEST) -i
GOFMT=gofmt -w

all:
	$(GOINSTALL)
	$(GOTEST)
	$(GOBUILD)

clean: 
	$(GOCLEAN)

build:
	$(GOCLEAN)
	$(GOBUILD)
	$(GOTEST)

cleanbuildinstall:
	$(GOCLEAN)
	$(GOBUILD)
	$(GOINSTALL)