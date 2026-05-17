package hotspot

import (
	"context"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
)

const cookiePath = "/ip/hotspot/cookie"

// CookieList → /ip/hotspot/cookie/print (analisis §1.8).
func (c *Client) CookieList(ctx context.Context) ([]domain.HotspotCookie, error) {
	reply, err := c.dev.Path(cookiePath).Print().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToCookies(reply.Rows), nil
}

// CookieCount → /ip/hotspot/cookie/print count-only="" (analisis §1.8).
func (c *Client) CookieCount(ctx context.Context) (int, error) {
	reply, err := c.dev.Path(cookiePath).Print().Count().Exec(ctx)
	if err != nil {
		return 0, err
	}
	return mikrotik.AtoiOr(reply.Raw.Done.Map["ret"], 0), nil
}

// CookieByUser → /ip/hotspot/cookie/print ?user=<name>.
func (c *Client) CookieByUser(ctx context.Context, user string) ([]domain.HotspotCookie, error) {
	if user == "" {
		return nil, mikrotik.ErrInvalidArgument
	}
	reply, err := c.dev.Path(cookiePath).Print().Where("user", user).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return sentencesToCookies(reply.Rows), nil
}

// CookieRemove → /ip/hotspot/cookie/remove (analisis §1.8).
func (c *Client) CookieRemove(ctx context.Context, id string) error {
	if id == "" {
		return mikrotik.ErrInvalidArgument
	}
	_, err := c.dev.Path(cookiePath).Remove(ctx, id)
	return err
}

func sentencesToCookies(rows []*roslib.Sentence) []domain.HotspotCookie {
	out := make([]domain.HotspotCookie, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.HotspotCookie{
			ID:      s.Get(".id"),
			User:    s.Get("user"),
			Domain:  s.Get("domain"),
			Expires: s.Get("expires-in"),
		})
	}
	return out
}
