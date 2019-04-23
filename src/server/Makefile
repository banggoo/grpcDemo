BIN=server

all: ${BIN}

${BIN}:
	go generate server.go
	go build ${LDFLAGS} -o ${BIN} server.go

install:
	go install ${LDFLAGS} -o ${BIN} server.go

clean:
	if [ -f ${BIN} ]; then rm ${BIN}; fi

.PHONY: clean install
