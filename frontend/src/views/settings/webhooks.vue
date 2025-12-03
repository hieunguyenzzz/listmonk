<template>
  <div>
    <div class="items webhooks">
      <div class="block box" v-for="(item, n) in data.webhooks" :key="n">
        <div class="columns">
          <div class="column is-2">
            <b-field :label="$t('globals.buttons.enabled')">
              <b-switch v-model="item.enabled" name="enabled" :native-value="true" />
            </b-field>
            <b-field>
              <a @click.prevent="$utils.confirm(null, () => removeWebhook(n))" href="#" class="is-size-7">
                <b-icon icon="trash-can-outline" size="is-small" />
                {{ $t('globals.buttons.delete') }}
              </a>
            </b-field>
          </div><!-- first column -->

          <div class="column" :class="{ disabled: !item.enabled }">
            <div class="columns">
              <div class="column is-4">
                <b-field :label="$t('globals.fields.name')" label-position="on-border"
                  :message="$t('settings.webhooks.nameHelp')">
                  <b-input v-model="item.name" name="name" placeholder="mywebhook" :maxlength="200" />
                </b-field>
              </div>
              <div class="column is-8">
                <b-field :label="$t('settings.webhooks.url')" label-position="on-border"
                  :message="$t('settings.webhooks.urlHelp')">
                  <b-input v-model="item.url" name="url" placeholder="https://example.com/webhook"
                    :maxlength="500" expanded type="url" pattern="https?://.*" />
                </b-field>
              </div>
            </div><!-- name/url -->

            <div class="columns">
              <div class="column">
                <b-field :label="$t('settings.webhooks.secret')" label-position="on-border"
                  :message="$t('settings.webhooks.secretHelp')">
                  <b-input v-model="item.secret" name="secret" type="password"
                    :placeholder="$t('globals.messages.passwordChange')" :maxlength="200" />
                </b-field>
              </div>
            </div><!-- secret -->
            <hr />

            <div class="columns">
              <div class="column">
                <b-field :label="$t('settings.webhooks.events')" label-position="on-border"
                  :message="$t('settings.webhooks.eventsHelp')">
                  <b-taginput v-model="item.events" :data="filteredEvents" autocomplete
                    :allow-new="false" :open-on-focus="true" field="label" placeholder="Select events"
                    @typing="filterEvents">
                    <template #default="props">
                      <span>{{ props.option }}</span>
                    </template>
                    <template #empty>
                      No events found
                    </template>
                  </b-taginput>
                </b-field>
              </div>
            </div><!-- events -->
            <hr />

            <div class="columns">
              <div class="column is-6">
                <b-field :label="$t('settings.webhooks.maxConns')" label-position="on-border"
                  :message="$t('settings.webhooks.maxConnsHelp')">
                  <b-numberinput v-model="item.max_conns" name="max_conns" type="is-light" controls-position="compact"
                    placeholder="5" min="1" max="100" />
                </b-field>
              </div>
              <div class="column is-6">
                <b-field :label="$t('settings.webhooks.timeout')" label-position="on-border"
                  :message="$t('settings.webhooks.timeoutHelp')">
                  <b-input v-model="item.timeout" name="timeout" placeholder="5s" :pattern="regDuration"
                    :maxlength="10" />
                </b-field>
              </div>
            </div>
          </div>
        </div><!-- second container column -->
      </div><!-- block -->
    </div><!-- webhooks -->

    <b-button @click="addWebhook" icon-left="plus" type="is-primary">
      {{ $t('globals.buttons.addNew') }}
    </b-button>
  </div>
</template>

<script>
import Vue from 'vue';
import { regDuration } from '../../constants';

const allEvents = [
  'subscriber.created',
  'subscriber.updated',
  'subscriber.unsubscribed',
  'subscriber.confirmed',
  'subscriber.blocklisted',
  'subscriber.deleted',
  'campaign.started',
  'campaign.finished',
  'tracking.link_click',
  'tracking.email_open',
  'tracking.bounce',
];

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
      regDuration,
      allEvents,
      filteredEvents: allEvents,
    };
  },

  methods: {
    addWebhook() {
      this.data.webhooks.push({
        uuid: '',
        enabled: true,
        name: '',
        url: '',
        secret: '',
        events: [...allEvents],
        max_conns: 5,
        timeout: '5s',
      });

      this.$nextTick(() => {
        const items = document.querySelectorAll('.webhooks input[name="name"]');
        items[items.length - 1].focus();
      });
    },

    removeWebhook(i) {
      this.data.webhooks.splice(i, 1);
    },

    filterEvents(text) {
      this.filteredEvents = allEvents.filter((e) => e.toLowerCase().includes(text.toLowerCase()));
    },
  },
});
</script>
