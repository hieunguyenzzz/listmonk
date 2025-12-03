-- autoresponders
-- name: get-autoresponders-for-list
-- Get active autoresponder campaigns for a specific list.
SELECT c.*,
    COALESCE(templates.body, (SELECT body FROM templates WHERE is_default = true LIMIT 1), '') AS template_body
FROM campaigns c
LEFT JOIN templates ON templates.id = c.template_id
JOIN campaign_lists cl ON cl.campaign_id = c.id
WHERE c.type = 'autoresponder'
    AND c.status = 'running'
    AND cl.list_id = $1;

-- name: check-autoresponder-sent
-- Check if an autoresponder has already been sent to a subscriber for a specific list.
SELECT EXISTS(
    SELECT 1 FROM autoresponder_history
    WHERE campaign_id = $1 AND subscriber_id = $2 AND list_id = $3
);

-- name: record-autoresponder-sent
-- Record that an autoresponder was sent to a subscriber.
INSERT INTO autoresponder_history (campaign_id, subscriber_id, list_id)
VALUES ($1, $2, $3)
ON CONFLICT (campaign_id, subscriber_id, list_id) DO NOTHING;
