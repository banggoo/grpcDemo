BIN=client

all: ${BIN}

${BIN}:
	go build ${LDFLAGS} -o ${BIN} client.go

install:
	go install ${LDFLAGS} -o ${BIN} client.go

clean:
	if [ -f ${BIN} ]; then rm ${BIN}; fi

.PHONY: clean install
