package gather

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/kurin/blazer/b2"
	"github.com/rwcarlsen/goexif/exif"
)

// Gather holds information about our gathering
type Gather struct {
	ctx        context.Context
	b2client   *b2.Client
	b2bucket   *b2.Bucket
	b2uri      *url.URL
	searchDirs []string
	basedir    string
}

type thumbnailSpec struct {
	width  int
	height int
	// keep vertical pictures vertical? (or make them all horiontal)
	preserveOrientation bool
}

func (g *Gather) find(name string) (string, error) {
	for _, dir := range g.searchDirs {
		// we check if it's available
		filename := filepath.Join(dir, name)

		exists, err := fileExists(filename)
		if err != nil {
			return "", err
		}
		if exists {
			return filename, nil
		}
	}
	return "", nil
}

func (g *Gather) findOrDownload(object *b2.Object) (string, error) {
	name := object.Name()
	filename, err := g.find(name)
	if err != nil {
		return "", err
	}
	if filename != "" {
		return filename, nil
	}
	dir := filepath.Join(g.basedir, "download")
	filename = filepath.Join(dir, name)
	err = b2DownloadFile(g.ctx, g.b2bucket, object, filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func (g *Gather) findThumbnail(name string, size thumbnailSpec) (string, error) {
	filename := g.thumbnailFilenameFor(name, size)
	exists, err := fileExists(filename)
	if err != nil {
		return "", err
	}
	if exists {
		return filename, nil
	}
	return "", nil
}

func (g *Gather) thumbnailFilenameFor(name string, size thumbnailSpec) string {
	sizeDir := fmt.Sprintf("%dx%d", size.width, size.height)
	return filepath.Join(g.basedir, "thumbnails", sizeDir, name)
}

func (g *Gather) createThumbnail(object *b2.Object, size thumbnailSpec) (string, error) {
	filename, err := g.findOrDownload(object)
	if err != nil {
		return "", err
	}
	thumbnailFilename := g.thumbnailFilenameFor(object.Name(), size)
	err = makeThumbnail(filename, thumbnailFilename, size)
	if err != nil {
		return "", err
	}
	return thumbnailFilename, nil
}

func (g *Gather) yay() error {
	b2prefix := g.b2uri.Path[1:]

	log.Printf("will use prefix %s\n", b2prefix)

	var cursor = &b2.Cursor{
		Prefix: b2prefix,
	}
	for {
		log.Printf("listing objects with Prefix %s!\n", cursor.Prefix)
		objs, next, err := g.b2bucket.ListObjects(g.ctx, 1000, cursor)
		if err != nil && err != io.EOF {
			return err
		}
		for _, obj := range objs {

			// We might already have the file here...

			// TODO: loop around available locations

			name := obj.Name()

			ext := strings.ToLower(filepath.Ext(name))

			if ext != ".jpg" {
				log.Printf("ignoring entry with extension [%s]\n", ext)
				continue
			}

			/*
				b2attrs, err := obj.Attrs(g.ctx)
				if err != nil {
					return err
				}
			*/

			log.Printf("[%s] processing\n", name)

			// Ensure we have a thumbnail

			thumbnailSpecs := []thumbnailSpec{
				thumbnailSpec{1024, 768, true},
				thumbnailSpec{320, 240, false},
			}

			for _, size := range thumbnailSpecs {
				thumbnailFilename, err := g.findThumbnail(name, size)
				if err != nil {
					return err
				}

				if thumbnailFilename == "" {
					// No thumbnail :( make one!
					thumbnailFilename, err = g.createThumbnail(obj, size)
					if err != nil {
						return err
					}
					log.Printf("[%s] created %dx%d thumbnail\n", name, size.width, size.height)
				}
			}

			// Ensure we have metadata

			metadataName := name + ".json"

			metadataFilename := filepath.Join(g.basedir, "metadata", metadataName)

			metadataExists, err := fileExists(metadataFilename)
			if err != nil {
				return err
			}

			if !metadataExists {
				otherPlace, err := g.find(metadataName)
				if err != nil {
					return err
				}
				if otherPlace == "" {
					log.Printf("cannot find metadata for %s\n", name)
					// better download it...
					// ...except I haven't uploaded any to b2 yet
					continue
				}
				err = copyFile(otherPlace, metadataFilename)
				if err != nil {
					return err
				}
				log.Printf("[%s] copied meta\n", name)
			}

			// log.Printf("metadata is %s\n", metadataFilename)

			// return nil

			/*

					SHA1 := calculateSHA1(filename)
					stat, err := os.Stat(filename)
					if err != nil {
						return err
					}

					if SHA1 != b2attrs.SHA1 {
						log.Printf("SHA1 does not match :/ %s vs %s\n", SHA1, b2attrs.SHA1)
						log.Fatal("dead!")
					} else {
						log.Printf("SHA1 matches!\n")
					}

					if stat.Size() != b2attrs.Size {
						log.Printf("size does not match :/ %d vs %d\n", stat.Size(), b2attrs.Size)
						log.Fatal("dead!")
					} else {
						log.Printf("size matches!\n")
					}

				err = os.MkdirAll(filepath.Dir(thumbnailFilename), os.ModePerm)
				if err != nil {
					return err
				}

				err = makeThumbnail(filename, thumbnailFilename)
				if err != nil {
					return err
				}
			*/
			// log.Printf("created thumbnail %", thumbnailFilename)
		}
		if err == io.EOF {
			return nil
		}

		if next == nil {
			break
		}

		cursor = next
	}
	return nil
}

// NewGather creates a gather struct with everything you need
func NewGather(b2id string, b2key string, b2path string) (*Gather, error) {

	ctx := context.Background()

	b2client, err := b2.NewClient(ctx, b2id, b2key)
	if err != nil {
		return nil, err
	}
	b2uri, err := url.Parse(b2path)
	if err != nil {
		return nil, err
	}

	b2bucket, err := b2client.Bucket(ctx, b2uri.Host)
	if err != nil {
		return nil, err
	}

	// destDirBase := "/tmp/pictures-foo"
	// metadataDirBase := filepath.Join(destDirBase, "metadata")
	// downloadDirBase := filepath.Join(destDirBase, "downloads")
	// thumbnailDirBase := filepath.Join(destDirBase, "thumbnails", "300x300")

	basedir := "/tmp/pictures-foo"

	return &Gather{
		ctx:      ctx,
		b2client: b2client,
		b2bucket: b2bucket,
		b2uri:    b2uri,
		basedir:  basedir,
		searchDirs: []string{
			filepath.Join(basedir, "downloads"),
			"/Pictures",
			"/run/media/nick/hank/Pictures",
		},
	}, nil
}

// Run gathers stuff
func Run(b2id string, b2key string, b2path string) error {

	gather, err := NewGather(b2id, b2key, b2path)
	if err != nil {
		return err
	}

	return gather.yay()
}

func b2DownloadFile(ctx context.Context, bucket *b2.Bucket, b2object *b2.Object, destFilename string) error {

	log.Printf("[%s] downloading from b2\n", b2object.Name())

	downloadDir := filepath.Dir(destFilename)
	err := os.MkdirAll(downloadDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	reader := b2object.NewReader(ctx)
	defer reader.Close()

	// destFile, err := b2.file.Create(destFilename)
	destFile, err := os.OpenFile(destFilename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	// reader.ConcurrentDownloads = downloads
	if _, err := io.Copy(destFile, reader); err != nil {
		destFile.Close()
		return err
	}
	return destFile.Close()
}

func makeThumbnail(sourceFilename string, destFilename string, size thumbnailSpec) error {

	orientation, err := getOrientation(sourceFilename)
	if err != nil {
		return err
	}

	src, err := imaging.Open(sourceFilename)
	if err != nil {
		return nil
	}

	// http://sylvana.net/jpegcrop/exif_orientation.html
	if orientation == 6 {
		src = imaging.Rotate270(src)
	} else if orientation == 3 {
		src = imaging.Rotate180(src)
	} else if orientation == 8 {
		src = imaging.Rotate90(src)
	}

	srcWidth := src.Bounds().Dx()
	srcHeight := src.Bounds().Dy()

	width := 0
	height := 0

	if size.preserveOrientation {

		longest := 0
		shortest := 0

		if size.width > size.height {
			longest = size.width
			shortest = size.height
		} else {
			longest = size.height
			shortest = size.width
		}

		if srcWidth > srcHeight {
			width = longest
			height = shortest
		} else {
			width = shortest
			height = longest
		}

	} else {

		width = size.width
		height = size.height

	}

	dst := imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)

	destDir := filepath.Dir(destFilename)
	err = os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	err = imaging.Save(dst, destFilename)
	if err != nil {
		return err
	}
	return nil
}

func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func calculateSHA1(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func copyFile(source string, dest string) error {
	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer from.Close()

	destDir := filepath.Dir(dest)
	err = os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return err
	}

	to, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

func getOrientation(filename string) (int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}

	x, err := exif.Decode(f)
	if err != nil {
		return 0, err
	}
	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return 0, err
	}
	return tag.Int(0)
}
