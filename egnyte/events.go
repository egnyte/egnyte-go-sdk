package egnyte

import (
	"context"
	"fmt"
)

// Our Events API supports polling for changes based on a cursor
// which indicates a position in the sequence of events that have occurred on your domain
func (c *Client) EventCursor(ctx context.Context) (*EventID, error) {
	uri := URI_FETCH_EVENT_ID
	url := fmt.Sprintf("%s", c.root)
	reqOpts := &requestOptions{
		Method: "GET",
		Root:   url,
		Path:   uri,
	}

	var event *EventID
	_, err := c.doRequest(ctx, reqOpts, nil, &event)
	if err != nil {
		return nil, err
	}
	return event, nil
}
