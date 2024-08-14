package zenodo

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"quantify.earth/reclaimer/internal/utils"
)

type ZenodoCreator struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
}

type ZenodoRecordMetadata struct {
	Title           string            `json:"title"`
	DOI             string            `json:"doi"`
	PublicationData string            `json:"publication_date"`
	Description     string            `json:"description"`
	AccessRight     string            `json:"access_right"`
	Creators        []ZenodoCreator   `json:"creators"`
	License         map[string]string `json:"license"`
	Notes           string            `json:"notes"`
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

func FetchRecord(zenodoID string) (ZenodoRecord, error) {
	url := fmt.Sprintf("https://zenodo.org/api/records/%s", zenodoID)
	resp, err := utils.HTTPGet(url, nil)
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

func FetchData(zenodoID string, filename string, extract bool, output string) error {

	record, err := FetchRecord(zenodoID)
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

	tmpdir, err := os.MkdirTemp("", "zenodo-*")
	if nil != err {
		return fmt.Errorf("failed to make temp dir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	tempDownloadPath := path.Join(tmpdir, targetFilename)
	out, err := os.Create(tempDownloadPath)
	if nil != err {
		return fmt.Errorf("failed to create temp download file: %w", err)
	}

	resp, err := utils.HTTPGet(downloadURL, nil)
	if nil != err {
		out.Close()
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Close()
		return fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if nil != err {
		return fmt.Errorf("failed to download file: %w", err)
	}

	ext := path.Ext(targetFilename)

	if extract && (".zip" == ext) {

		zipReader, err := zip.OpenReader(tempDownloadPath)
		if nil != err {
			return fmt.Errorf("failed to open zip file: %w", err)
		}
		defer zipReader.Close()

		generatedFiles := []string{}
		for _, innerFile := range zipReader.File {
			rawDest := path.Join(tmpdir, innerFile.Name)
			dest := path.Clean(rawDest)
			if !strings.HasPrefix(dest, tmpdir) {
				return fmt.Errorf("uncompressing file escapes temp dir: %s", innerFile.Name)
			}

			if innerFile.FileInfo().IsDir() {
				err = os.MkdirAll(dest, os.ModePerm)
				if nil != err {
					return fmt.Errorf("failed to create explicit dir from zip %s: %w", innerFile.Name, err)
				}
			} else {
				dir := path.Dir(dest)
				err = os.MkdirAll(dir, os.ModePerm)
				if nil != err {
					return fmt.Errorf("failed to create implicit dir from zip %s: %w", innerFile.Name, err)
				}

				out, err := os.Create(dest)
				if nil != err {
					return fmt.Errorf("failed to create file for extracted data %s: %w", dest, err)
				}
				defer out.Close()
				compress, err := innerFile.Open()
				if nil != err {
					return fmt.Errorf("failed to open file for extracted data %s: %w", innerFile.Name, err)
				}
				defer compress.Close()

				_, err = io.Copy(out, compress)
				if nil != err {
					return fmt.Errorf("failed to copy data %s: %w", innerFile.Name, err)
				}

				generatedFiles = append(generatedFiles, innerFile.Name)
			}
		}

		// put everything in the final place
		if (1 == len(generatedFiles)) && ("" != output) {
			destinationPath, err := utils.MakeOutputPath(generatedFiles[0], output)
			if nil != err {
				return fmt.Errorf("failed to make output path: %w", err)
			}

			err = utils.MoveFileByPath(tempDownloadPath, destinationPath)
			if nil != err {
				return fmt.Errorf("failed to move result to %s: %w", destinationPath, err)
			}
		} else {

			// Treat output as directory
			if "" == output {
				cwd, err := os.Getwd()
				if nil != err {
					return fmt.Errorf("failed to look up cwd: %w", err)
				}
				output = cwd
			}

			for _, generatedFileName := range generatedFiles {
				sourcePath := path.Join(tmpdir, generatedFileName)
				destinationPath := path.Join(output, generatedFileName)
				destinationDirectory := path.Dir(destinationPath)
				err = os.MkdirAll(destinationDirectory, os.ModePerm)
				if nil != err {
					return fmt.Errorf("failed to make output directory %s: %w", output, err)
				}

				err = utils.MoveFileByPath(sourcePath, destinationPath)
				if nil != err {
					return fmt.Errorf("failed to move result to %s: %w", destinationPath, err)
				}
			}
		}

	} else {
		if extract {
			fmt.Fprintf(os.Stderr, "warning: ignoring extract argument as extension '%s' doesn't match\n", ext)
		}

		destinationPath, err := utils.MakeOutputPath(targetFilename, output)
		if nil != err {
			return fmt.Errorf("failed to make output path: %w", err)
		}

		err = utils.MoveFileByPath(tempDownloadPath, destinationPath)
		if nil != err {
			return fmt.Errorf("failed to move result to %s: %w", destinationPath, err)
		}
	}

	return nil
}

func inspect(zenodoID string) error {
	record, err := FetchRecord(zenodoID)
	if nil != err {
		return err
	}

	fmt.Printf("title: %s\n", record.Title)
	fmt.Printf("creators:\n")
	for _, creator := range record.Metadata.Creators {
		fmt.Printf("\t%s, %s\n", creator.Name, creator.Affiliation)
	}
	if len(record.Metadata.License) > 0 {
		fmt.Printf("license:\n")
		for key, value := range record.Metadata.License {
			fmt.Printf("\t%s: %s\n", key, value)
		}
	}
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
		filename = flag.String("filename", "", "Specific item within resource to download. If ommitted download first item.")
		extract  = flag.Bool("extract", false, "If item is compressed extract automatically")
		output   = flag.String("output", "", "Destination name (filename for single item, directory name if multiple)")
	)
	flag.Parse(args)

	if (nil == zenodoID) || (nil == filename) || (nil == extract) || (nil == output) {
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
	if "" == *filename {
		err = inspect(*zenodoID)
	} else {
		err = FetchData(*zenodoID, *filename, *extract, *output)
	}
	if nil != err {
		fmt.Fprintf(os.Stderr, "ERROR: %v", err)
		os.Exit(1)
	}
}
