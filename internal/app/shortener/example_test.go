package shortener

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func ExampleShortener_APIShortenURL() {
	request := `{"url":"http://google.com"}`
	result := struct {
		Result string `json:"result"`
	}{}
	resp, err := http.Post("localhost:8080/api/shorten", "application/json", strings.NewReader(request))
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		// handle error
	}
	fmt.Printf("Short url for http://google.com: %s", result.Result)
}

func ExampleShortener_ShortenURL() {
	url := "http://youtube.com"
	resp, err := http.Post("localhost:8080/api/shorten", "text/plain", strings.NewReader(url))
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}
	fmt.Printf("Short url for http://youtube.com: %s", body)
}

func ExampleShortener_BatchShortenURL() {
	request := `
	[
		{
			"correlation_id": "Yandex",
			"original_url": "http://yandex.ru"
		},
		{
			"correlation_id": "Github",
			"original_url": "https://github.com/vanamelnik"
		}
	]`
	got := []struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}{}

	resp, err := http.Post("localhost:8080", "application/json", strings.NewReader(request))
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&got)
	if err != nil {
		// handle error
	}

	fmt.Println("Short URLs:")
	for _, res := range got {
		fmt.Printf("%s:\t%s\n", res.CorrelationID, res.ShortURL)
	}
}

func ExampleShortener_DeleteURLs() {
	reqBody := `["a2b38tjg", "8sfj93bf"]` // Ключи для удаления
	c := http.Client{}
	r, err := http.NewRequest(http.MethodDelete, "localhost:8080", strings.NewReader(reqBody))
	if err != nil {
		// handle error
	}
	resp, err := c.Do(r)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
}
