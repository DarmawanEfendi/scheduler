package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type IGCS interface {
	Close()
	ReadAllObjects(bucket string, limit int) (map[string][]byte, error)
	DeleteObject(bucketName string, filename string) error
	UploadObject(bucketName string, filename string, data []byte) error
	MoveObject(srcBucketName string, destBucketName string, filename string) error
}

// docker run -d --name fake-gcs-server -p 4443:4443 -v ${PWD}/files:/data fsouza/fake-gcs-server -backend memory -scheme http -public-host gcs:4443 -external-url http://gcs:4443
// https://github.com/fsouza/fake-gcs-server/issues/280
type GCS struct {
	client *storage.Client
	ctx    context.Context
}

func New(isRunEmulator bool) (*GCS, error) {
	options := make([]option.ClientOption, 0)
	if isRunEmulator {
		options = append(options, option.WithEndpoint("http://fake-gcs-server:4443/storage/v1/"))
		options = append(options, option.WithoutAuthentication())
		options = append(options, option.WithHTTPClient(&http.Client{
			Transport: &HostFixRoundTripper{&http.Transport{}},
		}))
	}

	ctx := context.Background()
	// Creates a client.
	client, err := storage.NewClient(ctx, options...)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, fmt.Errorf("failed to create gcs client: %v", err.Error())
	}

	return &GCS{
		client: client,
		ctx:    ctx,
	}, nil
}

type HostFixRoundTripper struct {
	Proxy http.RoundTripper
}

func (l HostFixRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Host = "gcs:4443"
	res, err := l.Proxy.RoundTrip(request)
	if res != nil {
		location := res.Header.Get("Location")
		if len(location) != 0 {
			res.Header.Set("Location", strings.Replace(location, "gcs", "localhost", 1))
		}
	}
	return res, err
}

func (g *GCS) Close() {
	g.client.Close()
}

func (g *GCS) getBucketObjects(bucket string, size int) ([]string, error) {
	ctx, cancel := context.WithTimeout(g.ctx, time.Second*10)
	defer cancel()

	it := g.client.Bucket(bucket).Objects(ctx, nil)

	filenames := make([]string, 0)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		filenames = append(filenames, attrs.Name)
		if size != -1 && len(filenames) > size {
			break
		}
	}
	return filenames, nil
}

func (g *GCS) ReadAllObjects(bucketName string, limit int) (map[string][]byte, error) {
	filenames, err := g.getBucketObjects(bucketName, limit)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, filename := range filenames {
		rc, err := g.client.Bucket(bucketName).Object(filename).NewReader(g.ctx)
		if err != nil {
			return nil, fmt.Errorf("readFile: unable to open file from bucket %q, file %q: %v", bucketName, filename, err)
		}

		bytes, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("readFile: unable to read data from bucket %q, file %q: %v", bucketName, filename, err)
		}
		result[filename] = bytes

		rc.Close()
	}
	return result, nil
}

func (g *GCS) DeleteObject(bucketName string, filename string) error {
	err := g.client.Bucket(bucketName).Object(filename).Delete(g.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (g *GCS) UploadObject(bucketName string, filename string, data []byte) error {
	ctx, cancel := context.WithTimeout(g.ctx, time.Second*50)
	defer cancel()

	wc := g.client.Bucket(bucketName).Object(filename).NewWriter(ctx)
	wc.ChunkSize = 0

	if _, err := io.Copy(wc, bytes.NewBuffer(data)); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	// Data can continue to be added to the file until the writer is closed.
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	return nil
}

func (g *GCS) MoveObject(srcBucketName string, destBucketName string, filename string) error {
	ctx, cancel := context.WithTimeout(g.ctx, time.Second*10)
	defer cancel()

	src := g.client.Bucket(srcBucketName).Object(filename)
	dst := g.client.Bucket(destBucketName).Object(filename)

	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return fmt.Errorf("Object(%q).CopierFrom(%q.json).Run: %v", destBucketName, filename, err)
	}
	if err := src.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q.json).Delete: %v", filename, err)
	}
	return nil
}
