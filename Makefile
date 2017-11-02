DEPS := github.com/gordonklaus/portaudio
DEPS += github.com/mjibson/go-dsp
DEPS += github.com/Arafatk/glot

.PHONY: all clean commands test

all: commands test run

clean:
	rm -fr vendor
	make -C cmd/gn clean
	
vendor:
	mkdir -p vendor
	for repo in $(DEPS); do git clone https://$$repo vendor/$$repo; done
	rm -fr vendor/*/*/*/.git
	
commands:
	make -C cmd/gn

test: 
	make -C cmd/gn test

run: 
	./cmd/gn/gn --quiet=false
