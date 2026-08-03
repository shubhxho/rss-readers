package feed

import (
	"io"
	"net/http"
)

// maxFeedBytes caps a single feed download to guard against runaway responses.
const maxFeedBytes = 16 << 20 // 16 MiB

func readAllLimited(resp *http.Response) ([]byte, error) {
	return io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
}
