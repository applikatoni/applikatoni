package deploy

import (
	"log"
)

const (
	ASCII_RESET = "\x1b[0m"

	ASCII_BLACK   = "\x1b[30m"
	ASCII_RED     = "\x1b[31m"
	ASCII_GREEN   = "\x1b[32m"
	ASCII_YELLOW  = "\x1b[33m"
	ASCII_BLUE    = "\x1b[34m"
	ASCII_MAGENTA = "\x1b[35m"
	ASCII_CYAN    = "\x1b[36m"
	ASCII_WHITE   = "\x1b[37m"
)

func ConsoleLogger(logs <-chan LogEntry) {
	for entry := range logs {
		switch entry.EntryType {
		case COMMAND_START:
			log.Printf("%s -- %sSTARTING:%s %s", entry.Origin, ASCII_YELLOW, ASCII_RESET, entry.Message)
		case COMMAND_STDOUT_OUTPUT:
			log.Printf("%s -- STDOUT -- %s", entry.Origin, entry.Message)
		case COMMAND_STDERR_OUTPUT:
			log.Printf("%s -- %sSTDERR%s -- %s", entry.Origin, ASCII_CYAN, ASCII_RESET, entry.Message)
		case COMMAND_FAIL:
			log.Printf("%s -- %sFAILED:%s %s", entry.Origin, ASCII_RED, ASCII_RESET, entry.Message)
		case COMMAND_SUCCESS:
			log.Printf("%s -- %sSUCCESS:%s %s", entry.Origin, ASCII_GREEN, ASCII_RESET, entry.Message)

		case STAGE_START:
			log.Printf("%sSTARTING STAGE: %s%s", ASCII_YELLOW, entry.Message, ASCII_RESET)
		case STAGE_FAIL:
			log.Printf("%sSTAGE FAILED:%s %s", ASCII_RED, ASCII_RESET, entry.Message)
		case STAGE_SUCCESS:
			log.Printf("%sSTAGE SUCCESS: %s%s", ASCII_GREEN, entry.Message, ASCII_RESET)
		case STAGE_RESULT:
			log.Printf("%sSTAGE RESULT: %s%s", ASCII_YELLOW, entry.Message, ASCII_RESET)

		case DEPLOYMENT_START:
			log.Printf("%sSTARTING DEPLOYMENT: %s%s", ASCII_MAGENTA, entry.Message, ASCII_RESET)
		case DEPLOYMENT_FAIL:
			log.Printf("%sDEPLOYMENT FAILED: %s%s", ASCII_RED, entry.Message, ASCII_RESET)
		case DEPLOYMENT_SUCCESS:
			log.Printf("%sDEPLOYMENT FINISHED: %s%s", ASCII_MAGENTA, entry.Message, ASCII_RESET)

		case KILL_RECEIVED:
			log.Printf("%sKILL RECEIVED: %s%s", ASCII_RED, entry.Message, ASCII_RESET)
		}
	}
}
