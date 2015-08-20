package deploy

import (
	"bufio"
	"io"
)

func streamPipe(pipe io.Reader, target chan string) {
	reader := bufio.NewReader(pipe)

	for {
		line, err := reader.ReadBytes('\n')
		if s := string(line); s != "" {
			target <- s
		}
		if err != nil {
			break
		}
	}
}
