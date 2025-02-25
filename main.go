package main

import (
	"fmt"
	"os"
)

const usage = `sharec
Usage: sharec <command> [arguments]

Commands:
  help, -h, --help  to get this menu
  offer             to create a new offer
  connect <id>      to connect to an offer
`

func parse(command string) {
  if (command == "help" || command == "--help" || command == "-h") {
    fmt.Print(usage);
    return;
  }
  if (command == "offer") {
    fmt.Println("offer");
    return;
  }
  if (command == "connect") {
    if (len(os.Args) < 3) {
      fmt.Println("you need to provide an offer id. sharec connect <id>");
      return;
    }
    fmt.Println("connect");
    return;
  } 
  if (command == "help"){
    fmt.Print(usage)
  } else {
    fmt.Println("Error: invalid command. use --help flag for more details");
    return;
  }

}

func main() {
	if len(os.Args) < 2 {
    fmt.Print(usage)
    return;
	}

	command := os.Args[1];
  parse(command);
}
