package api

import (
	"fmt"
	"html"
	"strings"

	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
)

func hookUploader(requestData *RequestData, apiBase string) error {
	torrentData, err := fetchResponseData(requestData, requestData.TorrentID, "torrent", apiBase)
	if err != nil {
		return err
	}

	username := strings.ToLower(torrentData.Response.Torrent.Username)
	usernames := parseAndTrimList(requestData.Uploaders)

	log.Trace().Msgf("[%s] Requested uploaders [%s]: %s", requestData.Indexer, requestData.Mode, strings.Join(usernames, ", "))

	isListed := stringInSlice(username, usernames)
	if (requestData.Mode == "blacklist" && isListed) || (requestData.Mode == "whitelist" && !isListed) {
		log.Debug().Msgf("[%s] Uploader (%s) is not allowed", requestData.Indexer, username)
		return fmt.Errorf("uploader is not allowed")
	}
	return nil
}

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

func hookSize(requestData *RequestData, apiBase string) error {
	torrentData, err := fetchResponseData(requestData, requestData.TorrentID, "torrent", apiBase)
	if err != nil {
		return err
	}

	torrentSize := bytesize.ByteSize(torrentData.Response.Torrent.Size)

	log.Trace().Msgf("[%s] Torrent size: %s, Requested size range: %s - %s", requestData.Indexer, torrentSize, requestData.MinSize, requestData.MaxSize)

	if (requestData.MinSize != 0 && torrentSize < requestData.MinSize) ||
		(requestData.MaxSize != 0 && torrentSize > requestData.MaxSize) {
		log.Debug().Msgf("[%s] Torrent size %s is outside the requested size range: %s to %s", requestData.Indexer, torrentSize, requestData.MinSize, requestData.MaxSize)
		return fmt.Errorf("torrent size is outside the requested size range")
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
