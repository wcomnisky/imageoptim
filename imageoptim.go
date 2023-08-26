package imageoptim

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	apiUser string
}

type Option struct {
	value string
}

// ImageOptim options. See https://imageoptim.com/api/post
var (
	Full       = Option{"full"}
	Fit        = Option{"fit"}
	ScaleDown  = Option{"scale-down"}
	QualityLow = Option{"quality=low"}
)

func NewClient(apiUser string) *Client {
	return &Client{apiUser}
}

func (c *Client) OptimizeImage(options []Option, input string) ([]byte, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		requestURL := c.createURLForURL(options, input)
		return c.processRequest(requestURL, nil, "")
	}

	return c.processLocalFile(options, input)
}

func (c *Client) createURLForURL(options []Option, imageURL string) string {
	var optStrings []string
	for _, opt := range options {
		optStrings = append(optStrings, opt.value)
	}
	return fmt.Sprintf("https://im2.io/%s/%s/%s", c.apiUser, strings.Join(optStrings, ","), imageURL)
}

func (c *Client) processLocalFile(options []Option, imagePath string) ([]byte, error) {
	requestURL := c.createURLForLocal(options)

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fw, err := w.CreateFormFile("file", filepath.Base(imagePath))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return nil, err
	}
	w.Close()

	return c.processRequest(requestURL, &b, w.FormDataContentType())
}

func (c *Client) createURLForLocal(options []Option) string {
	var optStrings []string
	for _, opt := range options {
		optStrings = append(optStrings, opt.value)
	}
	return fmt.Sprintf("https://im2.io/%s/%s", c.apiUser, strings.Join(optStrings, ","))
}

func (c *Client) processRequest(requestURL string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest("POST", requestURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from ImageOptim: %s", string(bodyBytes))
	}

	if warning, ok := resp.Header["Warning"]; ok {
		fmt.Println("Warning:", warning)
	}

	return io.ReadAll(resp.Body)
}

func WidthxHeight(width, height int) Option {
	return Option{fmt.Sprintf("%dx%d", width, height)}
}

func Quality(preset string) Option {
	return Option{fmt.Sprintf("quality=%s", preset)}
}

func Format(name string) Option {
	return Option{fmt.Sprintf("format=%s", name)}
}

func Timeout(seconds int) Option {
	return Option{fmt.Sprintf("timeout=%d", seconds)}
}
