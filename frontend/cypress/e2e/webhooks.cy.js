const apiUrl = Cypress.env('apiUrl');

describe('Webhooks Settings', () => {
  it('Opens settings page', () => {
    cy.resetDB();
    cy.loginAndVisit('/admin/settings');
  });

  it('Opens webhooks settings tab', () => {
    // Find and click the Webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Verify we're on the webhooks tab.
    cy.get('.tab-item:visible').should('exist');
  });

  it('Adds a new webhook endpoint', () => {
    // Click "Add new" button.
    cy.get('.tab-item:visible button').contains('Add new').click();
    cy.wait(250);

    // Fill in webhook details.
    cy.get('.tab-item:visible .webhooks input[name="name"]').last().clear().type('test-webhook');
    cy.get('.tab-item:visible .webhooks input[name="url"]').last().clear().type('https://example.com/webhook');
    cy.get('.tab-item:visible .webhooks input[name="secret"]').last().clear().type('my-test-secret');

    // The events should be pre-selected (all events by default).
    // Verify the taginput has some events.
    cy.get('.tab-item:visible .webhooks .taginput').should('exist');

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();
  });

  it('Verifies webhook was saved via API', () => {
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;

      expect(data.webhooks).to.exist;
      expect(data.webhooks.length).to.equal(1);
      expect(data.webhooks[0].name).to.equal('test-webhook');
      expect(data.webhooks[0].url).to.equal('https://example.com/webhook');
      expect(data.webhooks[0].enabled).to.equal(true);

      // Secret should be masked (all bullet characters).
      expect(data.webhooks[0].secret).to.match(/^•+$/);

      // Events should be set.
      expect(data.webhooks[0].events).to.be.an('array');
      expect(data.webhooks[0].events.length).to.be.greaterThan(0);
    });
  });

  it('Toggles webhook enabled/disabled', () => {
    // Navigate to webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Find and click the enable/disable switch.
    cy.get('.tab-item:visible .webhooks .b-switch').first().click();
    cy.wait(100);

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();

    // Verify webhook is now disabled.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data.webhooks[0].enabled).to.equal(false);
    });
  });

  it('Re-enables webhook', () => {
    // Navigate to webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Find and click the enable/disable switch to re-enable.
    cy.get('.tab-item:visible .webhooks .b-switch').first().click();
    cy.wait(100);

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();

    // Verify webhook is now enabled.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data.webhooks[0].enabled).to.equal(true);
    });
  });

  it('Edits webhook URL', () => {
    // Navigate to webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Edit the URL.
    const newUrl = 'https://newexample.com/webhook/endpoint';
    cy.get('.tab-item:visible .webhooks input[name="url"]').first().clear().type(newUrl);

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();

    // Verify URL was updated.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data.webhooks[0].url).to.equal(newUrl);
    });
  });

  it('Adds a second webhook endpoint', () => {
    // Navigate to webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Click "Add new" button.
    cy.get('.tab-item:visible button').contains('Add new').click();
    cy.wait(250);

    // Fill in second webhook details.
    cy.get('.tab-item:visible .webhooks input[name="name"]').last().clear().type('second-webhook');
    cy.get('.tab-item:visible .webhooks input[name="url"]').last().clear().type('https://second.example.com/hook');

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();

    // Verify two webhooks exist.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data.webhooks.length).to.equal(2);
      expect(data.webhooks[1].name).to.equal('second-webhook');
    });
  });

  it('Deletes a webhook endpoint', () => {
    // Navigate to webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Click delete on the second webhook.
    cy.get('.tab-item:visible .webhooks a').contains('Delete').last().click();

    // Confirm deletion in modal.
    cy.get('.modal button.is-primary').click();
    cy.wait(250);

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();

    // Verify only one webhook remains.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data.webhooks.length).to.equal(1);
      expect(data.webhooks[0].name).to.equal('test-webhook');
    });
  });

  it('Deletes all webhooks', () => {
    // Navigate to webhooks tab.
    cy.get('.b-tabs nav a').contains('Webhooks').click();
    cy.wait(250);

    // Delete the remaining webhook.
    cy.get('.tab-item:visible .webhooks a').contains('Delete').click();
    cy.get('.modal button.is-primary').click();
    cy.wait(250);

    // Save settings.
    cy.get('[data-cy=btn-save]').click();
    cy.wait(500);

    cy.waitForBackend();

    // Verify no webhooks remain.
    cy.request(`${apiUrl}/api/settings`).should((response) => {
      const { data } = response.body;
      expect(data.webhooks.length).to.equal(0);
    });
  });
});
