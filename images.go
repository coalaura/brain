package brain

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/coalaura/lock"
	"github.com/revrost/go-openrouter"
)

var (
	files  = lock.NewLockMap[string]()
	client = http.Client{
		Timeout: 5 * time.Second,
	}
)

func TouchFile(path string) {
	file, err := os.Create(path)
	if err != nil {
		return
	}

	file.Close()
}

func UrlBasename(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	return strings.ToLower(filepath.Base(u.Path))
}

func IsImage(uri string) bool {
	ext := filepath.Ext(UrlBasename(uri))

	return ext == ".jpg" || ext == ".jpeg" || ext == ".webp" || ext == ".png" || ext == ".gif"
}

func ImagePath(uri string) string {
	dir := filepath.Join(os.TempDir(), "brain-cache")

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	hash := md5.New()

	hash.Write([]byte(uri))

	return filepath.Join(dir, hex.EncodeToString(hash.Sum(nil)))
}

func LoadImage(uri string) (string, error) {
	path := ImagePath(uri)

	files.Lock(path)
	defer files.Unlock(path)

	var reader io.Reader

	if _, err := os.Stat(path); os.IsNotExist(err) {
		resp, err := client.Get(uri)
		if err != nil {
			return "", err
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			TouchFile(path)

			return "", errors.New(resp.Status)
		}

		img, _, err := image.Decode(resp.Body)
		if err != nil {
			TouchFile(path)

			return "", err
		}

		file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return "", err
		}

		defer file.Close()

		var buf bytes.Buffer

		multi := io.MultiWriter(file, &buf)

		err = jpeg.Encode(multi, img, &jpeg.Options{
			Quality: 90,
		})
		if err != nil {
			defer os.Remove(path)

			return "", err
		}

		reader = &buf
	} else {
		file, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			return "", err
		}

		defer file.Close()

		reader = file
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	if len(b) < 64 {
		return "", errors.New("previously failed to load")
	}

	b64 := base64.StdEncoding.EncodeToString(b)

	return fmt.Sprintf("data:image/jpeg;base64,%s", b64), nil
}

func LoadImagePairs(pairs []openrouter.ChatMessagePart) []openrouter.ChatMessagePart {
	var (
		wg sync.WaitGroup
		mx sync.Mutex
	)

	for i, part := range pairs {
		if part.Type != openrouter.ChatMessagePartTypeImageURL {
			continue
		}

		wg.Go(func() {
			b64, err := LoadImage(part.ImageURL.URL)

			mx.Lock()
			defer mx.Unlock()

			if err != nil {
				pairs[i] = openrouter.ChatMessagePart{
					Type: openrouter.ChatMessagePartTypeText,
					Text: fmt.Sprintf("[Image failed to load: %s]", part.ImageURL.URL),
				}
			} else {
				pairs[i].ImageURL.URL = b64
			}
		})
	}

	wg.Wait()

	return pairs
}
