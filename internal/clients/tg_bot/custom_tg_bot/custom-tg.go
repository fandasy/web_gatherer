package custom_tg_bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"project/pkg/e"
	"strconv"
)

type Client struct {
	host     string
	basePath string
	client   http.Client
}

func New(host, token string) *Client {
	return &Client{
		host:     host,
		basePath: "bot" + token,
		client:   http.Client{},
	}
}

func (c *Client) LeaveChat(ctx context.Context, chatID int64) error {
	const fn = "custom_tg_bot.LeaveChat"

	q := url.Values{}
	q.Add("chat_id", strconv.FormatInt(chatID, 10))

	_, err := c.doRequest(ctx, "leaveChat", q)
	if err != nil {
		return e.Wrap(fn, err)
	}

	return nil
}

func (c *Client) doRequest(ctx context.Context, method string, query url.Values) ([]byte, error) {
	u := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   path.Join(c.basePath, method),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
