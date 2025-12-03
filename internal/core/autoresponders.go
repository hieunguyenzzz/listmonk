package core

import (
	"net/http"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetAutorespondersForList retrieves all active autoresponder campaigns for a given list.
func (c *Core) GetAutorespondersForList(listID int) ([]models.Campaign, error) {
	var out []models.Campaign
	if err := c.q.GetAutorespondersForList.Select(&out, listID); err != nil {
		c.log.Printf("error fetching autoresponders for list %d: %v", listID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "autoresponders"))
	}
	return out, nil
}

// HasReceivedAutoresponder checks if an autoresponder has already been sent to a subscriber
// for a specific list.
func (c *Core) HasReceivedAutoresponder(campaignID, subscriberID, listID int) (bool, error) {
	var exists bool
	if err := c.q.CheckAutoresponderSent.Get(&exists, campaignID, subscriberID, listID); err != nil {
		c.log.Printf("error checking autoresponder sent status: %v", err)
		return false, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "autoresponder"))
	}
	return exists, nil
}

// RecordAutoresponderSent records that an autoresponder was sent to a subscriber.
func (c *Core) RecordAutoresponderSent(campaignID, subscriberID, listID int) error {
	if _, err := c.q.RecordAutoresponderSent.Exec(campaignID, subscriberID, listID); err != nil {
		c.log.Printf("error recording autoresponder sent: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "autoresponder"))
	}
	return nil
}
