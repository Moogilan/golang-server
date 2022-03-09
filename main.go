package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var lastLines []string
var lastFileSize int64 = 0
var queueSize = 20
var filePath = "C:\\CheckOutLog\\CheckOutLog.txt"
var path = "C:\\CheckOutLog\\"

func enqueue(queue []string, element string) []string {
	queue = append(queue, element) // Simply append to enqueue.
	if len(queue) > queueSize {
		return queue[1:]
	}
	return queue
}

func initialRead() bool {

	lastLines = lastLines[:0]
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	lastFileSize := info.Size()
	log.Println("Read size :", lastFileSize)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		log.Println(scanner.Text())
		post(scanner.Text())
		lastLines = enqueue(lastLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return false
	}

	log.Println("Read done last lines :")
	log.Println("**********")
	for i := 0; i < len(lastLines); i++ {
		log.Println(lastLines[i])

	}
	log.Println("**********")
	return true
}

func post(rawdata string) {
	//values := map[string]string{"foo": "baz"}
	//jsonData, err := json.Marshal(values)

	req, err := http.NewRequest(http.MethodPost, "http://localhost:3000/cardaccessraw", strings.NewReader(rawdata))
	if err != nil {
		log.Fatal(err)
	}

	// appending to existing query args
	q := req.URL.Query()
	q.Add("foo", "bar")

	// assign encoded query string to http request
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Errored when sending request to the server")
		return
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Status)
	fmt.Println(string(responseBody))
}

func incrementalRead() bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	newFileSize := info.Size()
	log.Println("incremental file size ", newFileSize)
	if newFileSize < lastFileSize {
		log.Println("New smaller file, performs initial read")
		return initialRead()
	}
	if newFileSize == lastFileSize {
		log.Println("File not modified, exit")
		return true
	}
	if newFileSize > lastFileSize {
		log.Println("File Modified")
	}
	lastFileSize = newFileSize
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer file.Close()
	_, err = file.Seek(-400, 2)

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	//log.Println("pre new line >> " + scanner.Text())
	var newLineFound = false
	for scanner.Scan() {
		//log.Println("pre new line >> " + scanner.Text())
		if newLineFound == false {
			if isNewline(scanner.Text()) {
				newLineFound = true
				lastLines = enqueue(lastLines, scanner.Text())
				log.Println("new line >> " + scanner.Text())
			}
		} else {
			lastLines = enqueue(lastLines, scanner.Text())
			log.Println("new line >> " + scanner.Text())

		}

	}

	log.Println("")

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return false
	}

	//return true
	return true
}

func isNewline(inputLine string) bool {
	for i := 0; i < len(lastLines); i++ {
		if inputLine == lastLines[i] {
			return false
		}
	}
	return true
}

func main() {
	log.SetFlags(log.Ltime)
	for {
		if initialRead() == true {
			break
		}
		time.Sleep(5 * time.Second)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					if event.Name == filePath {
						incrementalRead()
					}
					log.Println("")
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
