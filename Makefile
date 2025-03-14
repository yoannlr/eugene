GO=/usr/local/go/bin/go

all: build man
man:
	go-md2man -in man.md -out eugene.1
build:
	$(GO) build
clean:
	rm eugene
install: build man
	cp eugene /usr/local/bin/
	mkdir -p /usr/local/share/man/man1
	cp eugene.1 /usr/local/share/man/man1/
