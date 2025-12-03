package main

import (
	"github.com/knadh/listmonk/internal/core"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/models"
)

// makeAutoresponderHook returns a callback that triggers autoresponder campaigns
// when subscribers are added to lists. This is plugged into the 'core' package.
func makeAutoresponderHook(co *core.Core, m *manager.Manager) func(sub models.Subscriber, listIDs []int, isConfirmation bool) error {
	return func(sub models.Subscriber, listIDs []int, isConfirmation bool) error {
		// Skip if subscriber is not enabled.
		if sub.Status != models.SubscriberStatusEnabled {
			return nil
		}

		for _, listID := range listIDs {
			// Get autoresponders for this list.
			camps, err := co.GetAutorespondersForList(listID)
			if err != nil {
				lo.Printf("error fetching autoresponders for list %d: %v", listID, err)
				continue
			}

			for _, camp := range camps {
				// Check trigger condition matches:
				// - If ar_trigger_on_confirm=true, only trigger on confirmation (isConfirmation=true)
				// - If ar_trigger_on_confirm=false, only trigger on subscription (isConfirmation=false)
				if camp.ARTriggerOnConfirm != isConfirmation {
					continue
				}

				// Check if already sent to this subscriber.
				sent, err := co.HasReceivedAutoresponder(camp.ID, sub.ID, listID)
				if err != nil {
					lo.Printf("error checking autoresponder status: %v", err)
					continue
				}
				if sent {
					continue
				}

				// Send the autoresponder.
				if err := m.SendAutoresponder(&camp, sub); err != nil {
					lo.Printf("error sending autoresponder '%s' to subscriber %d: %v", camp.Name, sub.ID, err)
					continue
				}

				// Record that we sent this autoresponder.
				if err := co.RecordAutoresponderSent(camp.ID, sub.ID, listID); err != nil {
					lo.Printf("error recording autoresponder sent: %v", err)
				}
			}
		}

		return nil
	}
}
