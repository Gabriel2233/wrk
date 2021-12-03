package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

type VideoInfo struct {
	VideoRenderer struct {
		VideoId string
		Title   struct {
			Runs []struct {
				Text string
			}
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Println("Error: expected one argument")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Printf("Error: Couldn't read file\n Cause: %s\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var videos []VideoInfo

	s := bufio.NewScanner(f)
	for s.Scan() {
		videoOptions, err := getContentsFromYt(s.Text())
		if err != nil {
			continue
		}

		indicies := presentOptions(videoOptions)
		userInput := awaitAnswerFromUser()
		index := indicies[userInput]

		videos = append(videos, videoOptions[index])
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error: Couldn't find home dir\n Cause: %s\n", err)
		os.Exit(1)
	}
	path := fmt.Sprintf("%s/Videos/wrk/", home)
	if exists, _ := exists(path); !exists {
		os.MkdirAll(path, 0755)
	}

	var wg sync.WaitGroup

	wg.Add(len(videos))
	for _, vd := range videos {
        go func(vd VideoInfo, wg *sync.WaitGroup) {
            outPath := fmt.Sprintf(`%s%s`, path, vd.VideoRenderer.Title.Runs[0].Text)
			cmd := exec.Command("yt-dlp", "-o", strings.ReplaceAll(outPath, "/", "-"), vd.VideoRenderer.VideoId)
			stdout, err := cmd.Output()
			if err != nil {
				fmt.Println(err.Error(), string(stdout))
			}

            wg.Done()
        }(vd, &wg)
	}
    wg.Wait()

	fmt.Println("Done")
}

func getContentsFromYt(searchQuery string) ([]VideoInfo, error) {
	query := strings.Join(strings.Split(searchQuery, " "), "+")
	url := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", query)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	form := strings.Join(strings.Split(string(body), "\n"), "")
	form = strings.Split(form, "var ytInitialData")[1]

	spot := strings.Split(form, "=")[1:]
	data := strings.Split(strings.Join(spot, "="), ";</script>")[0]

	var filteredVideoOptions []VideoInfo

	contents := gjson.Get(data, "contents.twoColumnSearchResultsRenderer.primaryContents.sectionListRenderer.contents.0.itemSectionRenderer.contents")
	err = json.Unmarshal([]byte(contents.String()), &filteredVideoOptions)
	if err != nil {
		return nil, err
	}

	return filteredVideoOptions, nil
}

func presentOptions(videoOptions []VideoInfo) map[int]int {
	ret := make(map[int]int, len(videoOptions))

	fmt.Printf("Choose one of the titles by their number: \n")

	idx := 0
	for i, opt := range videoOptions {
		if len(opt.VideoRenderer.Title.Runs) == 0 {
			continue
		}

		title := opt.VideoRenderer.Title.Runs[0].Text
		fmt.Printf(" %d ) - %s\n", idx+1, title)

		ret[idx+1] = i
		idx += 1
	}

	return ret
}

func awaitAnswerFromUser() int {
	s := bufio.NewScanner(os.Stdin)
	var ans int

	for s.Scan() {
		input, err := strconv.Atoi(s.Text())
		if err != nil {
			fmt.Println("Try again")
			continue
		} else {
			ans = input
			break
		}
	}

	return ans
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
