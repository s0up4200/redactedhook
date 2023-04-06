package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type RequestData struct {
	ID       int     `json:"id"`
	APIKey   string  `json:"apikey"`
	MinRatio float64 `json:"minratio"`
}

type ResponseData struct {
	Response struct {
		Stats struct {
			Ratio float64 `json:"ratio"`
		} `json:"stats"`
	} `json:"response"`
}

func main() {
	http.HandleFunc("/redacted/ratio", ratioCheckerHandler)
	log.Fatal(http.ListenAndServe(":42135", nil))
}

func ratioCheckerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Printf("Received request from %s", r.RemoteAddr)

	// Read JSON payload from the request body
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var requestData RequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := fmt.Sprintf("https://redacted.ch/ajax.php?action=user&id=%d", requestData.ID)

	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", requestData.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responseData ResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ratio := responseData.Response.Stats.Ratio
	minRatio := requestData.MinRatio

	if ratio < minRatio {
		w.WriteHeader(http.StatusIMUsed) // HTTP status code 226
		log.Printf("Returned ratio (%f) is below minratio (%f), responding with status 226\n", ratio, minRatio)
	} else {
		w.WriteHeader(http.StatusOK) // HTTP status code 200
		log.Printf("Returned ratio (%f) is equal to or above minratio (%f), responding with status 200\n", ratio, minRatio)
	}
}
