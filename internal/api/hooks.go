package api

import (
	"fmt"
	"html"
	"strings"

	"github.com/rs/zerolog/log"
)

func hookRecordLabel(requestData *RequestData, apiBase string) error {
	requestedRecordLabels := parseAndTrimList(requestData.RecordLabel)
	log.Trace().Msgf("[%s] Requested record labels: [%s]", requestData.Indexer, strings.Join(requestedRecordLabels, ", "))

	torrentData, err := fetchResponseData(requestData, requestData.TorrentID, "torrent", apiBase)
	if err != nil {
		return err
	}

	recordLabel := strings.ToLower(strings.TrimSpace(html.UnescapeString(torrentData.Response.Torrent.RecordLabel)))
	name := torrentData.Response.Group.Name

	if recordLabel == "" {
		log.Debug().Msgf("[%s] No record label found for release: %s", requestData.Indexer, name)
		return fmt.Errorf("record label not found")
	}

	if !stringInSlice(recordLabel, requestedRecordLabels) {
		log.Debug().Msgf("[%s] The record label '%s' is not included in the requested record labels: [%s]", requestData.Indexer, recordLabel, strings.Join(requestedRecordLabels, ", "))
		return fmt.Errorf("record label not allowed")
	}

	return nil
}

func hookRatio(requestData *RequestData, apiBase string) error {
	userID := getUserID(requestData)
	minRatio := requestData.MinRatio

	if userID == 0 || minRatio == 0 {
		if userID != 0 || minRatio != 0 {
			log.Warn().Msgf("[%s] Incomplete ratio check configuration: userID or minRatio is missing.", requestData.Indexer)
		}
		return nil
	}

	userData, err := fetchResponseData(requestData, userID, "user", apiBase)
	if err != nil {
		return err
	}

	ratio := userData.Response.Stats.Ratio
	username := userData.Response.Username

	log.Trace().Msgf("[%s] MinRatio set to %.2f for %s", requestData.Indexer, minRatio, username)

	if ratio < minRatio {
		log.Debug().Msgf("[%s] Returned ratio %.2f is below minratio %.2f for %s", requestData.Indexer, ratio, minRatio, username)
		return fmt.Errorf("returned ratio is below minimum requirement")
	}

	return nil
}

func parseAndTrimList(list string) []string {
	items := strings.Split(list, ",")
	for i, item := range items {
		items[i] = strings.ToLower(strings.TrimSpace(item))
	}
	return items
}

func stringInSlice(str string, list []string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

func getUserID(requestData *RequestData) int {
	if requestData.Indexer == "ops" {
		return requestData.OPSUserID
	}
	return requestData.REDUserID
}
