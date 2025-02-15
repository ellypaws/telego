package lib

import (
	"io"
	"log"
	"os"
)

func LogOutput(writer io.Writer) func() {
	logfile := `log.txt`
	f, _ := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	mw := io.MultiWriter(writer, f)
	r, w, _ := os.Pipe()

	os.Stdout = w
	os.Stderr = w

	log.SetOutput(mw)

	exit := make(chan bool)

	go func() {
		_, _ = io.Copy(mw, r)
		exit <- true
		close(exit)
	}()

	return func() {
		_ = w.Close()
		<-exit
		_ = f.Close()
	}
}
