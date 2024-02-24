package api

import (
	"fmt"
	"html"
	"strings"
	"math"
	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
	"github.com/pelletier/go-toml"
)

// checks if the uploader is allowed based on the requestData.
func hookUploader(requestData *RequestData, apiBase string) error {

	torrentData, err := fetchResponseData(requestData, requestData.TorrentID, "torrent", apiBase)
	if err != nil {
		return err
	}

	username := torrentData.Response.Torrent.Username
	usernames := strings.Split(requestData.Uploaders, ",")
	for i, uname := range usernames {
		usernames[i] = strings.TrimSpace(uname)
	}

	usernamesStr := strings.Join(usernames, ", ")
	log.Trace().Msgf("[%s] Requested uploaders [%s]: %s", requestData.Indexer, requestData.Mode, usernamesStr)

	isListed := false
	for _, uname := range usernames {
		if uname == username {
			isListed = true
			break
		}
	}

	if (requestData.Mode == "blacklist" && isListed) || (requestData.Mode == "whitelist" && !isListed) {
		log.Debug().Msgf("[%s] Uploader (%s) is not allowed", requestData.Indexer, username)
		return fmt.Errorf("uploader is not allowed")
	}
	return nil
}

// checks if the record label is allowed based on the requestData.
func hookRecordLabel(requestData *RequestData, apiBase string) error {
	torrentData, err := fetchResponseData(requestData, requestData.TorrentID, "torrent", apiBase)
	if err != nil {
		return err
	}

	recordLabel := strings.ToLower(strings.TrimSpace(html.UnescapeString(torrentData.Response.Torrent.RecordLabel)))
	name := torrentData.Response.Group.Name

	requestedRecordLabels := normalizeLabels(strings.Split(requestData.RecordLabel, ","))
	if recordLabel == "" {
		log.Debug().Msgf("[%s] No record label found for release: %s", requestData.Indexer, name)
		return fmt.Errorf("record label not allowed")
	}

	recordLabelsStr := strings.Join(requestedRecordLabels, ", ")
	log.Trace().Msgf("[%s] Requested record labels: [%s]", requestData.Indexer, recordLabelsStr)

	isRecordLabelPresent := contains(requestedRecordLabels, recordLabel)
	if !isRecordLabelPresent {
		log.Debug().Msgf("[%s] The record label '%s' is not included in the requested record labels: [%s]", requestData.Indexer, recordLabel, recordLabelsStr)
		return fmt.Errorf("record label not allowed")
	}

	return nil
}

// checks if the torrent size is within the allowed range based on the requestData.
func hookSize(requestData *RequestData, apiBase string) error {
	torrentData, err := fetchResponseData(requestData, requestData.TorrentID, "torrent", apiBase)
	if err != nil {
		return err
	}

	torrentSize := bytesize.ByteSize(torrentData.Response.Torrent.Size)
	minSize := bytesize.ByteSize(requestData.MinSize)
	maxSize := bytesize.ByteSize(requestData.MaxSize)

	log.Trace().Msgf("[%s] Torrent size: %s, Requested size range: %s - %s", requestData.Indexer, torrentSize, requestData.MinSize, requestData.MaxSize)

	if (requestData.MinSize != 0 && torrentSize < minSize) ||
		(requestData.MaxSize != 0 && torrentSize > maxSize) {
		log.Debug().Msgf("[%s] Torrent size %s is outside the requested size range: %s to %s", requestData.Indexer, torrentSize, minSize, maxSize)
		return fmt.Errorf("torrent size is outside the requested size range")
	}

	return nil

}

// checks if the user ratio is above the minimum requirement based on the requestData.
func hookRatio(requestData *RequestData, apiBase string) error {
	userID := requestData.REDUserID
	minRatio := requestData.MinRatio
	if requestData.Indexer == "ops" {
		userID = requestData.OPSUserID
	}

	// Check for incomplete configuration
	if userID == 0 || minRatio == 0 {
		if userID != 0 || minRatio != 0 {
			log.Warn().Msgf("[%s] Incomplete ratio check configuration: userID or minRatio is missing.", requestData.Indexer)
		}
		return nil // Exit early if either is zero, as the check cannot proceed
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

	// Check if dynamic_minratio toggle is enabled
    if config.DynamicRatio.Enabled {
        // Update minratio if the user's ratio is higher than the current minratio
        if ratio > minRatio {
            // Round the ratio down to 2 significant figures
            minRatio = math.Floor(ratio*100) / 100 // Round down to 2 decimal places
            // Update the minratio in the config file
            err := updateConfigMinRatio(requestData.Indexer, minRatio)
            if err != nil {
                log.Error().Msgf("[%s] Failed to update minratio in config file: %v", requestData.Indexer, err)
                return err
            }

            log.Info().Msgf("[%s] Updated minratio to %.2f for %s", requestData.Indexer, minRatio, username)
        }
    } else {
        log.Debug().Msgf("[%s] Dynamic minratio updating is disabled", requestData.Indexer)
    }

    return nil
}

// Function to update the minratio in the config file
func updateConfigMinRatio(configFilePath string, minRatio float64) error {
    // Load the existing config file
    config, err := toml.LoadFile(configFilePath)
    if err != nil {
        return err
    }

    // Update the minratio value under [dynamicratio] section
    config.Set("dynamicratio.enabled", minRatio)

    // Save the changes back to the config file
    err = config.WriteFile(configFilePath, 0644)
    if err != nil {
        return err
    }

    return nil
}
