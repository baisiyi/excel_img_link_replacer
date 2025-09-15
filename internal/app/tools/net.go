package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	_ "golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"
)

var (
	cdnHttpClient     *http.Client
	cdnHttpClientOnce sync.Once
)

// getCDNHTTPClient 返回一个为 CDN 访问优化的全局 HTTP 客户端（懒加载，线程安全）
func getCDNHTTPClient() *http.Client {
	cdnHttpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		cdnHttpClient = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("stopped after 10 redirects")
				}
				return nil
			},
		}
	})
	return cdnHttpClient
}

// GetCDNImageBytes 通过 CDN URL 获取内容（单个），返回内容字节
func GetCDNImageBytes(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, errors.New("url is empty")
	}
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return nil, fmt.Errorf("invalid url: %v", err)
	}

	client := getCDNHTTPClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	req.Header.Set("Accept", "image/jpeg,image/png,image/webp")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("bad status: %d", resp.StatusCode)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 仅支持 jpeg/jpg、png、webp，其它格式返回不支持
	if !(IsJPEGMagicNumber(body) || IsPNGMagicNumber(body) || IsWEBPMagicNumber(body)) {
		return nil, fmt.Errorf("unsupported image format")
	}

	// 解码并统一转为 PNG 返回
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("decode image failed: %v", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode png failed: %v", err)
	}
	return buf.Bytes(), nil
}

// BatchGetCDNImageBytes 批量获取 CDN 内容（并发）
// concurrency: 并发数量，<=0 使用默认 5
func BatchGetCDNImageBytes(ctx context.Context, urls []string, concurrency int) (map[string][]byte, error) {

	var resultMap sync.Map

	results := make(map[string][]byte)
	if len(urls) == 0 {
		return results, nil
	}
	urls = Unique(urls)

	if concurrency <= 0 {
		concurrency = 5
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(concurrency)

	for i := range urls {
		l := urls[i]
		eg.Go(func() error {
			body, err := GetCDNImageBytes(ctx, l)
			if err != nil {
				return nil
			}
			resultMap.Store(l, body)
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	resultMap.Range(func(key, value interface{}) bool {
		results[key.(string)] = value.([]byte)
		return true
	})

	return results, nil
}

// IsJPEGMagicNumber 检查 JPEG 魔数
func IsJPEGMagicNumber(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	// JPEG 文件以 0xFF 0xD8 开头
	return data[0] == 0xFF && data[1] == 0xD8
}

// IsPNGMagicNumber 检查 PNG 魔数
func IsPNGMagicNumber(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// PNG 文件以 89 50 4E 47 0D 0A 1A 0A 开头
	return data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A
}

// IsWEBPMagicNumber 检查 WebP 魔数（基于 RIFF + WEBP 头）
func IsWEBPMagicNumber(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	// 前 4 字节为 "RIFF"
	if !(data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F') {
		return false
	}
	// 第 8-11 字节为 "WEBP"
	return data[8] == 'W' && data[9] == 'E' && data[10] == 'B' && data[11] == 'P'
}

func Unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
