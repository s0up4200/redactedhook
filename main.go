package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RatioRequestData struct {
	ID       int     `json:"user_id"`
	APIKey   string  `json:"apikey"`
	MinRatio float64 `json:"minratio"`
}

type UploaderRequestData struct {
	ID        int    `json:"torrent_id"`
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
	// Configure zerolog to use colored output
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})

	http.HandleFunc("/redacted/ratio", checkRatio)
	http.HandleFunc("/redacted/uploader", checkUploader)
	log.Info().Msg("Starting server on 127.0.0.1:42135")
	err := http.ListenAndServe("127.0.0.1:42135", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

func checkRatio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Debug().Msgf("Received request from %s", r.RemoteAddr)

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
		log.Debug().Msgf("Failed to unmarshal JSON payload: %s", err.Error())
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

	var responseData RatioResponseData
	err = json.Unmarshal(respBody, &responseData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ratio := responseData.Response.Stats.Ratio
	minRatio := requestData.MinRatio

	if ratio < minRatio {
		w.WriteHeader(http.StatusIMUsed) // HTTP status code 226
		log.Debug().Msgf("Returned ratio (%f) is below minratio (%f), responding with status 226", ratio, minRatio)
	} else {
		w.WriteHeader(http.StatusOK) // HTTP status code 200
		log.Debug().Msgf("Returned ratio (%f) is equal to or above minratio (%f), responding with status 200", ratio, minRatio)
	}
}

func checkUploader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Debug().Msg("Non-POST method received")
		http.Error(w, "Only POST method is supported", http.StatusBadRequest)
		return
	}

	// Log request received
	log.Debug().Msgf("Received request from %s", r.RemoteAddr)

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var requestData UploaderRequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		log.Debug().Msgf("Failed to unmarshal JSON payload: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	endpoint := fmt.Sprintf("https://redacted.ch/ajax.php?action=torrent&id=%d", requestData.ID)

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
		log.Error().Str("JSON response indicates a failure: %s\n", responseData.Error)
		http.Error(w, responseData.Error, http.StatusBadRequest)
		return
	}

	username := responseData.Response.Torrent.Username
	usernames := strings.Split(requestData.Usernames, ",")

	log.Debug().Msgf("Found uploader: %s", username) // Print the uploader's username

	for _, uname := range usernames {
		if uname == username {
			w.WriteHeader(http.StatusIMUsed + 1) // HTTP status code 226
			log.Debug().Msgf("Uploader (%s) is blacklisted, responding with status 226", username)
			return
		}
	}

	w.WriteHeader(http.StatusOK) // HTTP status code 200
	log.Debug().Msg("Uploader not in blacklist, responding with status 200")
}
