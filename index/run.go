package index

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/index/scorch"
)

// Run indexes things
func Run() error {
	basedir := "/tmp/pictures-foo/metadata"

	index, err := OpenIndex("db.bleve")
	if err != nil {
		return err
	}
	defer index.Close()

	batch := index.NewBatch()

	walk(basedir, func(path string) {
		log.Printf("got file %s\n", path)
		data, err := readMetadata(filepath.Join(basedir, path))
		if err != nil {
			log.Fatal(err)
		}
		name := strings.TrimSuffix(path, ".json")
		batch.Index(name, data)
	})
	log.Printf("writing batch!\n")
	err = index.Batch(batch)
	if err != nil {
		return err
	}
	log.Printf("wrote batch!\n")
	return nil
}

type walkResult func(path string)

// type metadata map[string]interface{}

type metadata struct {
	SourceFile         string  `json:"SourceFile"`
	ExifToolVersion    float64 `json:"ExifTool:ExifToolVersion"`
	FileName           string  `json:"File:FileName"`
	Directory          string  `json:"File:Directory"`
	FileSize           int64   `json:"File:FileSize"`
	Make               string  `json:"EXIF:Make"`
	Model              string  `json:"EXIF:Model"`
	Orientation        int     `json:"EXIF:Orientation"`
	ExposureTime       float64 `json:"EXIF:ExposureTime"`
	ISO                int64   `json:"EXIF:ISO"`
	DateTimeOriginal   string  `json:"EXIF:DateTimeOriginal"`
	MaxApertureValue   float64 `json:"EXIF:MaxApertureValue"`
	LensInfo           string  `json:"EXIF:LensInfo"`
	LensModel          string  `json:"EXIF:LensModel"`
	Rating             int     `json:"XMP:Rating"`
	Label              string  `json:"XMP:Label"`
	Subject            string  `json:"XMP:Subject"`
	Keywords           string  `json:"IPTC:Keywords"`
	SonyDateTime       string  `json:"MakerNotes:SonyDateTime"`
	FacesDetected      int     `json:"MakerNotes:FacesDetected"`
	AmbientTemperature float64 `json:"MakerNotes:AmbientTemperature"`
	BatteryTemperature float64 `json:"MakerNotes:BatteryTemperature"`
	BatteryLevel       int     `json:"MakerNotes:BatteryLevel"`
	Aperture           float64 `json:"Composite:Aperture"`
	//LensID             int     `json:"Composite:LensID"`
	Megapixels   float64 `json:"Composite:Megapixels"`
	ShutterSpeed float64 `json:"Composite:ShutterSpeed"`
}

// OpenIndex opens or creates index at this path
func OpenIndex(indexPath string) (bleve.Index, error) {
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		log.Printf("creating new index...")
		mapping := bleve.NewIndexMapping()
		index, err := bleve.NewUsing(indexPath, mapping, scorch.Name, bleve.Config.DefaultKVStore, map[string]interface{}{})
		if err != nil {
			return nil, err
		}
		return index, nil
	}
	return index, nil
}

// OpenIndexReadOnly opens read only
func OpenIndexReadOnly(indexPath string) (bleve.Index, error) {
	index, err := bleve.OpenUsing(indexPath, map[string]interface{}{
		"read_only": true,
	})
	if err != nil {
		return nil, err
	}
	return index, nil
}

func readMetadata(filename string) (metadata, error) {
	var data []metadata
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return metadata{}, err
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return metadata{}, err
	}
	return data[0], nil
}

func walk(dir string, onResult walkResult) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
			return err
		}
		if !info.IsDir() {
			onResult(strings.TrimPrefix(path, dir)[1:])
		}
		return nil
	})
}
