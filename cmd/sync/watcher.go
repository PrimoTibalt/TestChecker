package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type CreateTopic struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func AttachWatcher(path string, breaker chan bool) error {
	partnerurls, err := PartnerURLs()
	if err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	watcher.Add(path)
	go func(breaker chan bool) {
		defer watcher.Close()
		for {
			renamed := false
			select {
			case <-breaker:
				fmt.Println("Waiting for operation to complete")
				goto anchor
			case event := <-watcher.Events:
				if isHiddenName(filename(event.Name)) {
					continue
				}
				switch event.Op {
				case fsnotify.Create:
					createTopicModel := CreateTopic{}
					fmt.Println("Creating...")
					createTopicModel.Name = event.Name
					if renamed {
						fmt.Println("And filling...")
						f, err := os.ReadFile(event.Name)
						if err != nil {
							fmt.Println(err)
							continue
						}
						createTopicModel.Content = string(f)
					}
					body, err := json.Marshal(&createTopicModel)
					if err != nil {
						fmt.Printf("could not marshal %v\n", createTopicModel)
						continue
					}
					broadcast(partnerurls, "/createTopic", "application/json", body)
					fmt.Println("Finished creating")
				case fsnotify.Remove:
					fmt.Println("Removing...")
					broadcast(partnerurls, "/removeTopic", "text", []byte(event.Name))
					fmt.Println("Finished removing")
				case fsnotify.Rename:
					fmt.Println("Renaming...")
					broadcast(partnerurls, "/removeTopic", "text", []byte(event.Name))
					fmt.Println("Finished renaming")
				case fsnotify.Write:
					fmt.Println("Writing...")
					createTopicModel := CreateTopic{}
					createTopicModel.Name = event.Name
					f, err := os.ReadFile(event.Name)
					if err != nil {
						fmt.Println("could not read file " + event.Name)
						continue
					}
					createTopicModel.Content = string(f)
					body, err := json.Marshal(&createTopicModel)
					if err != nil {
						fmt.Println("could not marshal new changes")
						continue
					}
					broadcast(partnerurls, "/changeTopic", "application/json", body)
					fmt.Println("Finished writing")
				}
			case err = <-watcher.Errors:
				panic(err)
			}
		}
	anchor:
	}(breaker)

	return nil
}

// broadcast sends the same local change to every partner at once, so a slow or
// unreachable machine does not delay the rest of them.
func broadcast(partnerurls []string, endpoint string, contentType string, body []byte) {
	var waitgroup sync.WaitGroup
	for _, partnerurl := range partnerurls {
		waitgroup.Add(1)
		go func() {
			defer waitgroup.Done()
			response, err := http.DefaultClient.Post(
				partnerurl+endpoint,
				contentType,
				bytes.NewReader(body),
			)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				fmt.Println(partnerurl + endpoint + " answered with " + response.Status)
			}
		}()
	}
	waitgroup.Wait()
}

// filename is the topic name inside a watcher event, which reports full paths.
func filename(eventName string) string {
	if strings.ContainsRune(eventName, '/') {
		return path.Base(eventName)
	}
	return eventName
}
