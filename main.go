package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type RatioRequestData struct {
	ID       string `json:"user_id"`
	APIKey   string `json:"apikey"`
	MinRatio string `json:"minratio"`
}

type UploaderRequestData struct {
	ID        string `json:"id"`
	APIKey    string `json:"apikey"`
	Usernames string `json:"uploaders"`
}

type RatioResponseData struct {
	Response struct {
		Stats struct {
			Ratio float64 `json:"ratio"`
		} `json:"stats"`
	} `json:"response"`
}

type UploaderResponseData struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	Response struct {
		Torrent struct {
			Username string `json:"username"`
		} `json:"torrent"`
	} `json:"response"`
}

func main() {
	http.HandleFunc("/redacted/ratio", checkRatio)
	http.HandleFunc("/redacted/uploader", checkUploader)
	log.Fatal(http.ListenAndServe(":42135", nil))
}

func checkRatio(w http.ResponseWriter, r *http.Request) {
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

	var requestData RatioRequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := fmt.Sprintf("https://redacted.ch/ajax.php?action=user&id=%s", requestData.ID)

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

	var responseData RatioResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ratio := responseData.Response.Stats.Ratio
	minRatio, err := strconv.ParseFloat(requestData.MinRatio, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if ratio < minRatio {
		w.WriteHeader(http.StatusIMUsed) // HTTP status code 226
		log.Printf("Returned ratio (%f) is below minratio (%f), responding with status 226\n", ratio, minRatio)
	} else {
		w.WriteHeader(http.StatusOK) // HTTP status code 200
		log.Printf("Returned ratio (%f) is equal to or above minratio (%f), responding with status 200\n", ratio, minRatio)
	}
}

func checkUploader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Printf("Received request from %s", r.RemoteAddr)

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var requestData UploaderRequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := fmt.Sprintf("https://redacted.ch/ajax.php?action=torrent&id=%s", requestData.ID)

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

	var responseData UploaderResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check for a "failure" status in the JSON response
	if responseData.Status == "failure" {
		log.Printf("JSON response indicates a failure: %s\n", responseData.Error)
		http.Error(w, responseData.Error, http.StatusBadRequest)
		return
	}

	username := responseData.Response.Torrent.Username
	usernames := strings.Split(requestData.Usernames, ",")

	log.Printf("Found uploader: %s\n", username) // Print the uploader's username

	for _, uname := range usernames {
		if uname == username {
			w.WriteHeader(http.StatusIMUsed + 1) // HTTP status code 226
			log.Printf("Uploader (%s) is blacklisted, responding with status 226\n", username)
			return
		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Printf("Uploader not in blacklist, responding with status 200\n")
}
