package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/kurin/blazer/b2"
)

func main() {

	flag.Parse()

	b2id := os.Getenv("B2_ACCOUNT_ID")
	b2key := os.Getenv("B2_ACCOUNT_KEY")

	args := flag.Args()
	if len(args) != 1 {
		fmt.Printf("Pass path as arg, e.g. b2://my-bucket/some-path\n")
		return
	}
	path := args[0]

	ctx := context.Background()

	c, err := b2.NewClient(ctx, b2id, b2key)
	if err != nil {
		log.Fatal(err)
	}

	uri, err := url.Parse(path)
	if err != nil {
		log.Fatal(err)
	}

	bucket, err := c.Bucket(ctx, uri.Host)
	if err != nil {
		log.Fatal(err)
	}

	prefix := uri.Path[1:]

	log.Printf("will use prefix %s\n", prefix)

	var cursor = &b2.Cursor{
		Prefix: prefix,
		// Delimiter: "/",
	}
	for {
		log.Printf("listing objects with Prefix %s!\n", cursor.Prefix)
		objs, next, err := bucket.ListObjects(ctx, 1000, cursor)

		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		for _, obj := range objs {

			// We might already have the file here...

			// TODO: loop around available locations

			destDirBase := "/tmp/pictures-foo"
			downloadDirBase := filepath.Join(destDirBase, "downloads")
			thumbnailDirBase := filepath.Join(destDirBase, "thumbnails", "300x300")
			storageDirBase := "/Pictures"

			baseDirs := []string{downloadDirBase, storageDirBase, "/run/media/nick/hank/Pictures"}

			name := obj.Name()

			ext := strings.ToLower(filepath.Ext(name))

			if ext != ".jpg" {
				log.Printf("ignoring entry with extension [%s]\n", ext)
				continue
			}

			log.Printf("processing [%s]\n", name)

			// Check for thumbnail first

			thumbnailFilename := filepath.Join(thumbnailDirBase, name)

			if fileExists(thumbnailFilename) {
				log.Printf("thumbnail already exists [%s]\n", thumbnailFilename)
				continue
			}

			filename := ""

			for _, dir := range baseDirs {
				// we check if it's available
				checkFilename := filepath.Join(dir, name)
				if _, err := os.Stat(checkFilename); os.IsNotExist(err) {
					// does not exist ... continue
					continue
				}
				filename = checkFilename
			}

			if filename != "" {
				log.Printf("exists in %s\n", filename)
			} else {
				log.Printf("need to download %s\n", name)
				downloadDir := filepath.Join(downloadDirBase, filepath.Dir(name))
				err = os.MkdirAll(downloadDir, os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
				filename = filepath.Join(downloadDirBase, name)
				err := downloadFile(ctx, bucket, 1, obj, filename)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("downloaded %s to %s\n", name, filename)
			}

			err = os.MkdirAll(filepath.Dir(thumbnailFilename), os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}

			err = makeThumbnail(filename, thumbnailFilename)
			if err != nil {
				log.Fatal(err)
			}
			// log.Printf("created thumbnail %", thumbnailFilename)
		}
		if err == io.EOF {
			return
		}

		if next == nil {
			break
		}

		cursor = next
	}
}

func downloadFile(ctx context.Context, bucket *b2.Bucket, downloads int, object *b2.Object, destFilename string) error {
	reader := object.NewReader(ctx)
	defer reader.Close()

	// destFile, err := b2.file.Create(destFilename)
	destFile, err := os.OpenFile(destFilename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	reader.ConcurrentDownloads = downloads
	if _, err := io.Copy(destFile, reader); err != nil {
		destFile.Close()
		return err
	}
	return destFile.Close()
}

func makeThumbnail(sourceFilename string, destFilename string) error {
	src, err := imaging.Open(sourceFilename)
	if err != nil {
		return nil
	}
	dst := imaging.Fill(src, 300, 300, imaging.Center, imaging.Lanczos)
	err = imaging.Save(dst, destFilename)
	if err != nil {
		return err
	}
	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Fatal(err)
	}
	return true
}
