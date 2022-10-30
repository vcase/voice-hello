package main

import (
	"github.com/vcase/voice-servicetemplate/generated"
)

func main() {
	generated.Run(&additionServer{}, &multiplicationServer{}, &unaryServer{})
}
