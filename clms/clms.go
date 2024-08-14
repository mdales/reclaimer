package clms

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"quantify.earth/reclaimer/internal/utils"
)

type CLMSSearchBatch struct {
	ID    string `json:"@id"`
	First string `json:"first"`
	Last  string `json:"last"`
	Next  string `json:"next"`
}

type CLMSDownloadInfo struct {
	ID         string `json:"@id"`
	Name       string `json:"name"`
	Collection string `json:"collection"`
	// This one is multiformat - can be a string or a struct
	// FullFormat string `json:"full_format"`
	FullPath   string   `json:"full_path"`
	FullSource string   `json:"full_source"`
	Layers     []string `json:"layers"`
}

type CLMSSearchItem struct {
	ID          string                        `json:"@id"`
	Type        string                        `json:"@type"`
	UID         string                        `json:"UID"`
	Title       string                        `json:"title"`
	Description string                        `json:"description"`
	Downloads   map[string][]CLMSDownloadInfo `json:"dataset_download_information"`
	ReviewState string                        `json:"review_state"`
}

type CLMSSearch struct {
	ID    string           `json:"@id"`
	Batch CLMSSearchBatch  `json:"batching"`
	Items []CLMSSearchItem `json:"items"`
	Total int              `json:"items_total"`
}

const baseURL = "https://land.copernicus.eu/api/"
const searchPathTemplate = "%s@search?b_start=%d&portal_type=DataSet&metadata_fields=UID&metadata_fields=dataset_full_format&&metadata_fields=dataset_download_information"

func FetchIndex() ([]CLMSSearchItem, error) {

	items := make([]CLMSSearchItem, 0)

	url := fmt.Sprintf(searchPathTemplate, baseURL, 0)
	for {
		headers := map[string]string{
			"Accept": "application/json",
		}
		resp, err := utils.HTTPGet(url, headers)
		if nil != err {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			r, err := io.ReadAll(resp.Body)
			body := resp.Status
			if nil == err {
				body = string(r)
			}
			return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, body)
		}

		var res CLMSSearch
		err = json.NewDecoder(resp.Body).Decode(&res)
		if nil != err {
			return nil, fmt.Errorf("failed to decode response for %s: %w", url, err)
		}

		// Technically not needed
		if len(res.Items) == 0 {
			break
		}

		items = append(items, res.Items...)

		if res.Batch.Last == url {
			break
		}
		if res.Batch.Next == url {
			break
		}
		url = res.Batch.Next
		if "" == url {
			return nil, fmt.Errorf("got invald response: no next URL")
		}
	}

	return items, nil
}

func inspectAll() error {
	index, err := FetchIndex()
	if nil != err {
		return err
	}

	for _, item := range index {
		fmt.Printf("%s: %s\n", item.UID, item.Title)
	}

	return nil
}

func inspect(UID string) error {
	index, err := FetchIndex()
	if nil != err {
		return err
	}

	for _, item := range index {
		if item.UID == UID {
			fmt.Printf("title: %s\n", item.Title)
			fmt.Printf("description: %s\n", item.Description)

			if items, ok := item.Downloads["items"]; ok {
				for _, item := range items {
					fmt.Printf("\t%s: %s\n", item.Name, item.FullPath)
				}
			}

			break
		}
	}

	return nil
}

func CLMSMain(args []string) {
	flag := flag.NewFlagSet("clms", flag.ExitOnError)
	var (
		UID = flag.String("uid", "", "UID of resource")
	)
	flag.Parse(args)

	if nil == UID {
		// stop the static analyser being upset
		panic("Flags didn't work")
	}

	var err error
	if "" == *UID {
		err = inspectAll()
	} else {
		err = inspect(*UID)
	}
	if nil != err {
		fmt.Fprintf(os.Stderr, "ERROR: %v", err)
		os.Exit(1)
	}
}
