package zenodo

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"syscall"
	"time"
)

type ZenodoCreator struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
}

type ZenodoRecordMetadata struct {
	Title           string          `json:"title"`
	DOI             string          `json:"doi"`
	PublicationData string          `json:"publication_date"`
	Description     string          `json:"description"`
	AccessRight     string          `json:"access_right"`
	Creators        []ZenodoCreator `json:"creators"`
}

type ZenodoFile struct {
	ID       string            `json:"id"`
	Key      string            `json:"key"`
	Size     int64             `json:"size"`
	Checksum string            `json:"checksum"`
	Links    map[string]string `json:"links"`
}

type ZenodoRecord struct {
	Created    time.Time            `json:"created"`
	Modified   time.Time            `json:"modified"`
	Updated    time.Time            `json:"updated"`
	ID         int                  `json:"id"`
	Revision   int                  `json:"revision"`
	DOI        string               `json:"doi"`
	DOIURL     string               `json:"doi_url"`
	Metadata   ZenodoRecordMetadata `json:"metadata"`
	Version    string               `json:"version"`
	Title      string               `json:"title"`
	Links      map[string]string    `json:"links"`
	Files      []ZenodoFile         `json:"files"`
	Status     string               `json:"status"`
	Statistics map[string]int       `json:"stats"`
	State      string               `json:"state"`
	Submitted  bool                 `json:"submitted"`
}

func fetchRecord(zenodoID string) (ZenodoRecord, error) {
	url := fmt.Sprintf("https://zenodo.org/api/records/%s", zenodoID)
	resp, err := http.Get(url)
	if nil != err {
		return ZenodoRecord{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ZenodoRecord{}, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	var record ZenodoRecord
	err = json.NewDecoder(resp.Body).Decode(&record)
	return record, err
}

func Fetch(zenodoID string, filename string, extract bool, output string) error {

	record, err := fetchRecord(zenodoID)
	if nil != err {
		return fmt.Errorf("failed to look up zenodo record: %w", err)
	}

	if 0 == len(record.Files) {
		return fmt.Errorf("record has no files")
	}

	targetFilename := ""
	downloadURL := ""
	for _, file := range record.Files {
		if ("" != filename) && (filename != file.Key) {
			continue
		}
		if url, ok := file.Links["self"]; ok {
			targetFilename = path.Base(file.Key)
			downloadURL = url
			break
		}
	}
	if "" == downloadURL {
		return fmt.Errorf("no download URL found")
	}
	if "" == targetFilename {
		return fmt.Errorf("download has no name")
	}

	destinationPath := output
	if "" == destinationPath {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to find current dir: %w", err)
		}
		destinationPath = path.Join(cwd, targetFilename)
	}

	tmpdir, err := os.MkdirTemp("", "zenodo-*")
	if nil != err {
		return fmt.Errorf("failed to make temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	outPath := path.Join(tmpdir, targetFilename)
	out, err := os.Create(outPath)
	if nil != err {
		out.Close()
		return fmt.Errorf("failed to create temp download file: %w", err)
	}

	resp, err := http.Get(downloadURL)
	if nil != err {
		out.Close()
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if nil != err {
		out.Close()
		return fmt.Errorf("failed to download file: %w", err)
	}
	out.Close()

	err = os.Rename(outPath, destinationPath)
	if nil != err {
		if errors.Is(err, syscall.EXDEV) {
			out, err := os.Open(outPath)
			if nil != err {
				return fmt.Errorf("failed to open source when copying to destination: %w", err)
			}
			defer out.Close()

			final, err := os.Create(destinationPath)
			if nil != err {
				return fmt.Errorf("failed to open destination for final copy: %w", err)
			}
			defer final.Close()

			_, err = io.Copy(final, out)
			if nil != err {
				return fmt.Errorf("error copying result to final place: %w", err)
			}
			// in a normal move file you'd delete the source here, but we don't need to
			// as the tmp dir will be deleted
		} else {
			return fmt.Errorf("failed to move result to %s: %w", destinationPath, err)
		}
	}

	return nil
}

func Inspect(zenodoID string) error {
	record, err := fetchRecord(zenodoID)
	if nil != err {
		return err
	}

	fmt.Printf("title: %s\n", record.Title)
	fmt.Printf("files:\n")
	for _, file := range record.Files {
		units := []string{"b", "Kb", "Mb", "Gb", "Tb"}
		unitindex := 0
		count := float64(file.Size)
		for idx := 0; idx < (len(units) - 1); idx++ {
			if 1024.0 > count {
				break
			}
			count = count / 1024.0
			unitindex += 1
		}

		fmt.Printf("\t%s (%.1f %s)\n", file.Key, count, units[unitindex])
	}

	return nil
}

func ZenodoMain(args []string) {
	flag := flag.NewFlagSet("zenodo", flag.ExitOnError)
	var (
		zenodoID = flag.String("zenodo_id", "", "Zenodo ID of resource")
		inspect  = flag.Bool("inspect", false, "only list the files, don't download")
		filename = flag.String("filename", "", "Specific item within resource to download. If ommitted download first item.")
		extract  = flag.Bool("extract", false, "If item is compressed extract automatically")
		output   = flag.String("output", "", "Path where file should be stored")
	)
	flag.Parse(args)

	if (nil == zenodoID) || (nil == filename) || (nil == output) || (nil == extract) || (nil == inspect) {
		// stop the static analyser being upset
		panic("Flags didn't work")
	}

	// Input sanitisation
	if "" == *zenodoID {
		fmt.Fprintf(os.Stderr, "Zenodo ID is requred\n")
		flag.Usage()
		os.Exit(1)
	}

	var err error
	if *inspect {
		err = Inspect(*zenodoID)
	} else {
		err = Fetch(*zenodoID, *filename, *extract, *output)
	}
	if nil != err {
		fmt.Fprintf(os.Stderr, "ERROR: %v", err)
		os.Exit(1)
	}
}
